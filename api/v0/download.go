package v0

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/qinguoyi/osproxy/app/models"
	"github.com/qinguoyi/osproxy/app/pkg/base"
	"github.com/qinguoyi/osproxy/app/pkg/repo"
	"github.com/qinguoyi/osproxy/app/pkg/storage"
	"github.com/qinguoyi/osproxy/app/pkg/thirdparty"
	"github.com/qinguoyi/osproxy/app/pkg/utils"
	"github.com/qinguoyi/osproxy/app/pkg/web"
	"github.com/qinguoyi/osproxy/bootstrap"
	"github.com/qinguoyi/osproxy/bootstrap/plugins"
	"io"
	"net/http"
	"os"
	"path"
	"sync"
	"time"
)

/*
对象下载
*/

// DownloadHandler    下载数据
//
//	@Summary      下载数据
//	@Description  下载数据
//	@Tags         下载
//	@Accept       application/json
//	@Param        uid        query  string  true  "文件uid"
//	@Param        name       query  string  true  "文件名称"
//	@Param        online     query  string  true  "是否在线"
//	@Param        date       query  string  true  "链接生成时间"
//	@Param        expire     query  string  true  "过期时间"
//	@Param        bucket     query  string  true  "存储桶"
//	@Param        object     query  string  true  "存储名称"
//	@Param        signature  query  string  true  "签名"
//	@Produce      application/json
//	@Success      200  {object}  web.Response
//	@Router       /api/storage/v0/download [get]
func DownloadHandler(c *gin.Context) {
	// 校验参数
	uidStr := c.Query("uid")
	name := c.Query("name")
	online := c.Query("online")
	date := c.Query("date")
	expireStr := c.Query("expire")
	bucketName := c.Query("bucket")
	objectName := c.Query("object")
	signature := c.Query("signature")

	if online == "" {
		online = "1"
	}
	if !utils.Contains(online, []string{"0", "1"}) {
		web.ParamsError(c, "online参数有误")
		return
	}

	uid, err, errorInfo := base.CheckValid(uidStr, date, expireStr)
	if err != nil {
		web.ParamsError(c, errorInfo)
		return
	}
	if !base.CheckDownloadSignature(date, expireStr, bucketName, objectName, signature) {
		web.ParamsError(c, "签名校验失败")
		return
	}

	var meta *models.MetaDataInfo
	lgRedis := new(plugins.LangGoRedis).NewRedis()
	val, err := lgRedis.Get(context.Background(), fmt.Sprintf("%s-meta", uidStr)).Result()
	// key在redis中不存在
	if err == redis.Nil {
		lgDB := new(plugins.LangGoDB).Use("default").NewDB()
		meta, err = repo.NewMetaDataInfoRepo().GetByUid(lgDB, uid)
		if err != nil {
			lgLogger.WithContext(c).Error("下载数据，查询数据元信息失败")
			web.InternalError(c, "内部异常")
			return
		}
		// 写入redis
		b, err := json.Marshal(meta)
		if err != nil {
			lgLogger.WithContext(c).Warn("下载数据，写入redis失败")
		}
		lgRedis.SetNX(context.Background(), fmt.Sprintf("%s-meta", uidStr), b, 5*60*time.Second)
	} else {
		if err != nil {
			lgLogger.WithContext(c).Error("下载数据，查询redis失败")
			web.InternalError(c, "")
			return
		}
		var msg models.MetaDataInfo
		if err := json.Unmarshal([]byte(val), &msg); err != nil {
			lgLogger.WithContext(c).Error("下载数据，查询redis结果，序列化失败")
			web.InternalError(c, "")
			return
		}
		// 续期
		lgRedis.Expire(context.Background(), fmt.Sprintf("%s-meta", uidStr), 5*60*time.Second)
		meta = &msg
	}
	bucketName = meta.Bucket
	objectName = meta.StorageName
	fileSize := meta.StorageSize
	start, end := base.GetRange(c.GetHeader("Range"), fileSize)
	c.Writer.Header().Add("Content-Length", fmt.Sprintf("%d", end-start+1))
	if online == "0" {
		c.Writer.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", name))
	} else {
		c.Writer.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%s", name))
	}
	c.Writer.Header().Add("Content-Type", meta.ContentType)
	c.Writer.Header().Add("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, fileSize))
	c.Writer.Header().Set("Accept-Ranges", "bytes")
	if start == fileSize {
		c.Status(http.StatusOK)
		return
	}
	if end == fileSize-1 {
		c.Status(http.StatusOK)
	} else {
		c.Status(http.StatusPartialContent)
	}

	ch := make(chan []byte, 1024*1024*20)
	proxyFlag := false
	// local存储: 单文件上传完uid会删除, 大文件合并后会删除
	if bootstrap.NewConfig("").Local.Enabled {
		dirName := path.Join(utils.LocalStore, uidStr)
		// 不分片：单文件或大文件已合并
		if !meta.MultiPart {
			dirName = path.Join(utils.LocalStore, bucketName, objectName)
		}
		if _, err := os.Stat(dirName); os.IsNotExist(err) {
			proxyFlag = true
		}
	}
	if proxyFlag {
		// 不在本地，询问集群内其他服务并转发
		serviceList, err := base.NewServiceRegister().Discovery()
		if err != nil || serviceList == nil {
			lgLogger.WithContext(c).Error("发现其他服务失败")
			web.InternalError(c, "发现其他服务失败")
			return
		}
		var wg sync.WaitGroup
		var ipList []string
		ipChan := make(chan string, len(serviceList))
		for _, service := range serviceList {
			wg.Add(1)
			go func(ip string, port string, ipChan chan string, wg *sync.WaitGroup) {
				defer wg.Done()
				res, err := thirdparty.NewStorageService().Locate(utils.Scheme, ip, port, uidStr)
				if err != nil {
					fmt.Print(err.Error())
					return
				}
				ipChan <- res
			}(service.IP, service.Port, ipChan, &wg)
		}
		wg.Wait()
		close(ipChan)
		for re := range ipChan {
			ipList = append(ipList, re)
		}
		if len(ipList) == 0 {
			lgLogger.WithContext(c).Error("发现其他服务失败")
			web.InternalError(c, "发现其他服务失败")
			return
		}
		proxyIP := ipList[0]
		_, bodyData, _, err := thirdparty.NewStorageService().DownloadForward(c, utils.Scheme, proxyIP,
			bootstrap.NewConfig("").App.Port)
		if err != nil {
			lgLogger.WithContext(c).Error("下载转发失败")
			web.InternalError(c, err.Error())
			return
		}
		defer bodyData.Close()
		// 避免响应体全部读入内存，导致内存溢出问题
		buffer := new(bytes.Buffer)
		_, err = io.Copy(buffer, bodyData)
		if err != nil {
			lgLogger.WithContext(c).Error("转发下载数据发送失败")
			web.InternalError(c, "转发下载数据发送失败")
			return
		}
		data := buffer.Bytes()
		for len(data) > 0 {
			chunkSize := 1024 * 1024 // 每次读取 1MB 数据
			if len(data) < chunkSize {
				chunkSize = len(data)
			}
			ch <- data[:chunkSize]
			data = data[chunkSize:]
		}

		// 关闭 channel
		close(ch)
	} else {
		// local在本地 || 其他os
		if !meta.MultiPart {
			go func() {
				step := int64(1 * 1024 * 1024)
				for {
					if start >= end {
						close(ch)
						break
					}
					length := step
					if start+length > end {
						length = end - start + 1
					}
					data, err := storage.NewStorage().Storage.GetObject(bucketName, objectName, start, length)
					if err != nil && err != io.EOF {
						lgLogger.WithContext(c).Error(fmt.Sprintf("从对象存储获取数据失败%s", err.Error()))
					}
					ch <- data
					start += step
				}
			}()

			// 这种场景，会先从minio中获取全部数据，再流式传输，所以下载前会等待一下，但会把内存打爆
			//go func() {
			//	data, err := inner.NewStorage().Storage.GetObject(bucketName, objectName, start, end-start+1)
			//	if err != nil && err != io.EOF {
			//		lgLogger.WithContext(c).Error(fmt.Sprintf("从minio获取数据失败%s", err.Error()))
			//	}
			//	ch <- data
			//	close(ch)
			//}()

		} else {
			// 分片数据传输
			var multiPartInfoList []models.MultiPartInfo
			val, err := lgRedis.Get(context.Background(), fmt.Sprintf("%s-multiPart", uidStr)).Result()
			// key在redis中不存在
			if err == redis.Nil {
				lgDB := new(plugins.LangGoDB).Use("default").NewDB()
				if err := lgDB.Model(&models.MultiPartInfo{}).Where(
					"storage_uid = ? and status = ?", uid, 1).Order("chunk_num ASC").Find(&multiPartInfoList).Error; err != nil {
					lgLogger.WithContext(c).Error("下载数据，查询分片数据失败")
					web.InternalError(c, "查询分片数据失败")
					return
				}
				// 写入redis
				b, err := json.Marshal(multiPartInfoList)
				if err != nil {
					lgLogger.WithContext(c).Warn("下载数据，写入redis失败")
				}
				lgRedis.SetNX(context.Background(), fmt.Sprintf("%s-multiPart", uidStr), b, 5*60*time.Second)
			} else {
				if err != nil {
					lgLogger.WithContext(c).Error("下载数据，查询redis失败")
					web.InternalError(c, "")
					return
				}
				var msg []models.MultiPartInfo
				if err := json.Unmarshal([]byte(val), &msg); err != nil {
					lgLogger.WithContext(c).Error("下载数据，查询reids，结果序列化失败")
					web.InternalError(c, "")
					return
				}
				// 续期
				lgRedis.Expire(context.Background(), fmt.Sprintf("%s-multiPart", uidStr), 5*60*time.Second)
				multiPartInfoList = msg
			}

			if meta.PartNum != len(multiPartInfoList) {
				lgLogger.WithContext(c).Error("分片数量和整体数量不一致")
				web.InternalError(c, "分片数量和整体数量不一致")
				return
			}

			// 查找起始分片
			index, totalSize := int64(0), int64(0)
			var startP, lengthP int64
			for {
				if totalSize >= start {
					startP, lengthP = 0, multiPartInfoList[index].StorageSize
				} else {
					if totalSize+multiPartInfoList[index].StorageSize > start {
						startP, lengthP = start-totalSize, multiPartInfoList[index].StorageSize-(start-totalSize)
					} else {
						totalSize += multiPartInfoList[index].StorageSize
						index++
						continue
					}
				}
				break
			}
			var chanSlice []chan int
			for i := 0; i < utils.MultiPartDownload; i++ {
				chanSlice = append(chanSlice, make(chan int, 1))
			}

			chanSlice[0] <- 1
			j := 0
			for i := 0; i < utils.MultiPartDownload; i++ {
				go func(i int, startP_, lengthP_ int64) {
					for {
						// 当前块计算完后，需要等待前一个块合并到主哈希
						<-chanSlice[i]

						if index >= int64(meta.PartNum) {
							close(ch)
							break
						}
						if totalSize >= start {
							startP_, lengthP_ = 0, multiPartInfoList[index].StorageSize
						}
						totalSize += multiPartInfoList[index].StorageSize

						data, err := storage.NewStorage().Storage.GetObject(
							multiPartInfoList[index].Bucket,
							multiPartInfoList[index].StorageName,
							startP_,
							lengthP_,
						)
						if err != nil && err != io.EOF {
							lgLogger.WithContext(c).Error(fmt.Sprintf("从对象存储获取数据失败%s", err.Error()))
						}
						// 合并到主哈希
						ch <- data
						index++
						// 这里要注意适配chanSlice的长度
						if j == utils.MultiPartDownload-1 {
							j = 0
						} else {
							j++
						}
						chanSlice[j] <- 1
					}
				}(i, startP, lengthP)
			}
		}
	}

	// 在使用 Stream 响应时，需要在调用stream之前设置status
	c.Stream(func(w io.Writer) bool {
		defer func() {
			if err := recover(); err != nil {
				lgLogger.WithContext(c).Error(fmt.Sprintf("stream流式响应出错，%s", err))
			}
		}()
		data, ok := <-ch
		if !ok {
			return false
		}
		_, err := w.Write(data)
		if err != nil {
			lgLogger.WithContext(c).Error(fmt.Sprintf("写入http响应出错，%s", err.Error()))
			return false
		}
		return true
	})
	return
}
