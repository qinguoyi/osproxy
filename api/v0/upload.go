package v0

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/qinguoyi/ObjectStorageProxy/app/models"
	"github.com/qinguoyi/ObjectStorageProxy/app/pkg/base"
	"github.com/qinguoyi/ObjectStorageProxy/app/pkg/repo"
	"github.com/qinguoyi/ObjectStorageProxy/app/pkg/storage"
	"github.com/qinguoyi/ObjectStorageProxy/app/pkg/thirdparty"
	"github.com/qinguoyi/ObjectStorageProxy/app/pkg/utils"
	"github.com/qinguoyi/ObjectStorageProxy/app/pkg/web"
	"github.com/qinguoyi/ObjectStorageProxy/bootstrap"
	"github.com/qinguoyi/ObjectStorageProxy/bootstrap/plugins"
	"go.uber.org/zap"
	"io"
	"os"
	"path"
	"strconv"
	"sync"
	"time"
)

/*
对象上传
*/

// UploadSingleHandler    上传单个文件
//
//	@Summary		上传单个文件
//	@Description	上传单个文件
//	@Tags			上传
//	@Accept			multipart/form-data
//	@Param			file		formData	file	true	"上传的文件"
//	@Param			uid			query		string	true	"文件uid"
//	@Param			md5			query		string	true	"md5"
//	@Param			date		query		string	true	"链接生成时间"
//	@Param			expire		query		string	true	"过期时间"
//	@Param			signature	query		string	true	"签名"
//	@Produce		application/json
//	@Success		200	{object}	web.Response
//	@Router			/api/storage/v0/upload [put]
func UploadSingleHandler(c *gin.Context) {
	uidStr := c.Query("uid")
	md5 := c.Query("md5")
	date := c.Query("date")
	expireStr := c.Query("expire")
	signature := c.Query("signature")

	uid, err, errorInfo := base.CheckValid(uidStr, date, expireStr)
	if err != nil {
		web.ParamsError(c, errorInfo)
		return
	}

	if !base.CheckUploadSignature(date, expireStr, signature) {
		web.ParamsError(c, "签名校验失败")
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		web.ParamsError(c, fmt.Sprintf("解析文件参数失败，详情：%s", err))
		return
	}

	// 判断记录是否存在
	lgDB := new(plugins.LangGoDB).Use("default").NewDB()
	metaData, err := repo.NewMetaDataInfoRepo().GetByUid(lgDB, uid)
	if err != nil {
		web.NotFoundResource(c, "当前上传链接无效，uid不存在")
		return
	}

	dirName := path.Join(utils.LocalStore, uidStr)
	// 判断是否上传过，md5
	resumeInfo, err := repo.NewMetaDataInfoRepo().GetResumeByMd5(lgDB, []string{md5})
	if err != nil {
		lgLogger.WithContext(c).Error("查询文件是否已上传失败")
		web.InternalError(c, "")
		return
	}
	if len(resumeInfo) != 0 {
		now := time.Now()
		if err := repo.NewMetaDataInfoRepo().Updates(lgDB, uid, map[string]interface{}{
			"bucket":       resumeInfo[0].Bucket,
			"storage_name": resumeInfo[0].StorageName,
			"address":      resumeInfo[0].Address,
			"md5":          md5,
			"storage_size": resumeInfo[0].StorageSize,
			"multi_part":   false,
			"status":       1,
			"updated_at":   &now,
			"content_type": resumeInfo[0].ContentType,
		}); err != nil {
			lgLogger.WithContext(c).Error("上传完更新数据失败")
			web.InternalError(c, "上传完更新数据失败")
			return
		}
		if err := os.RemoveAll(dirName); err != nil {
			lgLogger.WithContext(c).Error(fmt.Sprintf("删除目录失败，详情%s", err.Error()))
			web.InternalError(c, fmt.Sprintf("删除目录失败，详情%s", err.Error()))
			return
		}
		// 首次写入redis 元数据
		lgRedis := new(plugins.LangGoRedis).NewRedis()
		metaCache, err := repo.NewMetaDataInfoRepo().GetByUid(lgDB, uid)
		if err != nil {
			lgLogger.WithContext(c).Error("上传数据，查询数据元信息失败")
			web.InternalError(c, "内部异常")
			return
		}
		b, err := json.Marshal(metaCache)
		if err != nil {
			lgLogger.WithContext(c).Warn("上传数据，写入redis失败")
		}
		lgRedis.SetNX(context.Background(), fmt.Sprintf("%s-meta", uidStr), b, 5*60*time.Second)

		web.Success(c, "")
		return
	}
	// 判断是否在本地
	if _, err := os.Stat(dirName); os.IsNotExist(err) {
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
		_, _, _, err = thirdparty.NewStorageService().UploadForward(c, utils.Scheme, proxyIP,
			bootstrap.NewConfig("").App.Port, uidStr, true)
		if err != nil {
			lgLogger.WithContext(c).Error("上传单文件，转发失败")
			web.InternalError(c, err.Error())
			return
		}
		web.Success(c, "")
		return
	}
	// 在本地
	fileName := path.Join(utils.LocalStore, uidStr, metaData.StorageName)
	out, err := os.Create(fileName)
	if err != nil {
		lgLogger.WithContext(c).Error("本地创建文件失败")
		web.InternalError(c, "本地创建文件失败")
		return
	}
	src, err := file.Open()
	if err != nil {
		lgLogger.WithContext(c).Error("打开本地文件失败")
		web.InternalError(c, "打开本地文件失败")
		return
	}
	if _, err = io.Copy(out, src); err != nil {
		lgLogger.WithContext(c).Error("请求数据存储到文件失败")
		web.InternalError(c, "请求数据存储到文件失败")
		return
	}
	// 校验md5
	md5Str, err := base.CalculateFileMd5(fileName)
	if err != nil {
		lgLogger.WithContext(c).Error(fmt.Sprintf("生成md5失败，详情%s", err.Error()))
		web.InternalError(c, err.Error())
		return
	}
	if md5Str != md5 {
		web.ParamsError(c, fmt.Sprintf("校验md5失败，计算结果:%s, 参数:%s", md5Str, md5))
		return
	}
	// 上传到minio
	contentType, err := base.DetectContentType(fileName)
	if err != nil {
		lgLogger.WithContext(c).Error("判断文件content-type失败")
		web.InternalError(c, "判断文件content-type失败")
		return
	}
	if err := storage.NewStorage().Storage.PutObject(metaData.Bucket, metaData.StorageName, fileName, contentType); err != nil {
		lgLogger.WithContext(c).Error("上传到minio失败")
		web.InternalError(c, "上传到minio失败")
		return
	}
	// 更新元数据
	now := time.Now()
	fileInfo, _ := os.Stat(fileName)
	if err := repo.NewMetaDataInfoRepo().Updates(lgDB, metaData.UID, map[string]interface{}{
		"md5":          md5Str,
		"storage_size": fileInfo.Size(),
		"multi_part":   false,
		"status":       1,
		"updated_at":   &now,
		"content_type": contentType,
	}); err != nil {
		lgLogger.WithContext(c).Error("上传完更新数据失败")
		web.InternalError(c, "上传完更新数据失败")
		return
	}
	_, _ = out.Close(), src.Close()

	if err := os.RemoveAll(dirName); err != nil {
		lgLogger.WithContext(c).Error(fmt.Sprintf("删除目录失败，详情%s", err.Error()))
		web.InternalError(c, fmt.Sprintf("删除目录失败，详情%s", err.Error()))
		return
	}

	// 首次写入redis 元数据
	lgRedis := new(plugins.LangGoRedis).NewRedis()
	metaCache, err := repo.NewMetaDataInfoRepo().GetByUid(lgDB, uid)
	if err != nil {
		lgLogger.WithContext(c).Error("上传数据，查询数据元信息失败")
		web.InternalError(c, "内部异常")
		return
	}
	b, err := json.Marshal(metaCache)
	if err != nil {
		lgLogger.WithContext(c).Warn("上传数据，写入redis失败")
	}
	lgRedis.SetNX(context.Background(), fmt.Sprintf("%s-meta", uidStr), b, 5*60*time.Second)

	web.Success(c, "")
	return
}

// UploadMultiPartHandler    上传分片文件
//
//	@Summary		上传分片文件
//	@Description	上传分片文件
//	@Tags			上传
//	@Accept			multipart/form-data
//	@Param			file		formData	file	true	"上传的文件"
//	@Param			uid			query		string	true	"文件uid"
//	@Param			md5			query		string	true	"md5"
//	@Param			chunkNum	query		string	true	"当前分片id"
//	@Param			date		query		string	true	"链接生成时间"
//	@Param			expire		query		string	true	"过期时间"
//	@Param			signature	query		string	true	"签名"
//	@Produce		application/json
//	@Success		200	{object}	web.Response
//	@Router			/api/storage/v0/upload/multi [put]
func UploadMultiPartHandler(c *gin.Context) {
	uidStr := c.Query("uid")
	md5 := c.Query("md5")
	chunkNumStr := c.Query("chunkNum")
	date := c.Query("date")
	expireStr := c.Query("expire")
	signature := c.Query("signature")

	uid, err, errorInfo := base.CheckValid(uidStr, date, expireStr)
	if err != nil {
		web.ParamsError(c, errorInfo)
		return
	}

	chunkNum, err := strconv.ParseInt(chunkNumStr, 10, 64)
	if err != nil {
		web.ParamsError(c, errorInfo)
		return
	}

	if !base.CheckUploadSignature(date, expireStr, signature) {
		web.ParamsError(c, "签名校验失败")
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		web.ParamsError(c, fmt.Sprintf("解析文件参数失败，详情：%s", err))
		return
	}

	// 判断记录是否存在
	lgDB := new(plugins.LangGoDB).Use("default").NewDB()
	metaData, err := repo.NewMetaDataInfoRepo().GetByUid(lgDB, uid)
	if err != nil {
		web.NotFoundResource(c, "当前上传链接无效，uid不存在")
		return
	}
	// 判断当前分片是否已上传
	var lgRedis = new(plugins.LangGoRedis).NewRedis()
	ctx := context.Background()
	createLock := base.NewRedisLock(&ctx, lgRedis, fmt.Sprintf("multi-part-%d-%d-%s", uid, chunkNum, md5))
	if flag, err := createLock.Acquire(); err != nil || !flag {
		lgLogger.WithContext(c).Error("上传多文件抢锁失败")
		web.InternalError(c, "上传多文件抢锁失败")
		return
	}
	partInfo, err := repo.NewMultiPartInfoRepo().GetPartInfo(lgDB, uid, chunkNum, md5)
	if err != nil {
		lgLogger.WithContext(c).Error("多文件上传，查询分片信息失败")
		web.InternalError(c, "内部异常")
		return
	}
	if len(partInfo) != 0 {
		web.Success(c, "")
		return
	}
	_, _ = createLock.Release()

	// 判断是否在本地
	dirName := path.Join(utils.LocalStore, uidStr)
	if _, err := os.Stat(dirName); os.IsNotExist(err) {
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
		_, _, _, err = thirdparty.NewStorageService().UploadForward(c, utils.Scheme, proxyIP,
			bootstrap.NewConfig("").App.Port, uidStr, false)
		if err != nil {
			lgLogger.WithContext(c).Error("多文件上传，转发失败")
			web.InternalError(c, err.Error())
			return
		}
		web.Success(c, "")
		return
	}

	// 在本地
	fileName := path.Join(utils.LocalStore, uidStr, fmt.Sprintf("%d_%d", uid, chunkNum))
	out, err := os.Create(fileName)
	if err != nil {
		lgLogger.WithContext(c).Error("本地创建文件失败")
		web.InternalError(c, "本地创建文件失败")
		return
	}
	defer func(out *os.File) {
		_ = out.Close()
	}(out)
	src, err := file.Open()
	if err != nil {
		lgLogger.WithContext(c).Error("打开本地文件失败")
		web.InternalError(c, "打开本地文件失败")
		return
	}
	if _, err = io.Copy(out, src); err != nil {
		lgLogger.WithContext(c).Error("请求数据存储到文件失败")
		web.InternalError(c, "请求数据存储到文件失败")
		return
	}
	// 校验md5
	md5Str, err := base.CalculateFileMd5(fileName)
	if err != nil {
		lgLogger.WithContext(c).Error(fmt.Sprintf("生成md5失败，详情%s", err.Error()))
		web.InternalError(c, err.Error())
		return
	}
	if md5Str != md5 {
		lgLogger.WithContext(c).Error(fmt.Sprintf("校验md5失败，计算结果:%s, 参数:%s", md5Str, md5))
		web.ParamsError(c, fmt.Sprintf("校验md5失败，计算结果:%s, 参数:%s", md5Str, md5))
		return
	}
	// 上传到minio
	contentType := "application/octet-stream"
	if err := storage.NewStorage().Storage.PutObject(metaData.Bucket, fmt.Sprintf("%d_%d", uid, chunkNum),
		fileName, contentType); err != nil {
		lgLogger.WithContext(c).Error("上传到minio失败")
		web.InternalError(c, "上传到minio失败")
		return
	}

	// 创建元数据
	now := time.Now()
	fileInfo, _ := os.Stat(fileName)
	if err := repo.NewMultiPartInfoRepo().Create(lgDB, &models.MultiPartInfo{
		StorageUid:   uid,
		ChunkNum:     int(chunkNum),
		Bucket:       metaData.Bucket,
		StorageName:  fmt.Sprintf("%d_%d", uid, chunkNum),
		StorageSize:  fileInfo.Size(),
		PartFileName: fmt.Sprintf("%d_%d", uid, chunkNum),
		PartMd5:      md5Str,
		Status:       1,
		CreatedAt:    &now,
		UpdatedAt:    &now,
	}); err != nil {
		lgLogger.WithContext(c).Error("上传完更新数据失败")
		web.InternalError(c, "上传完更新数据失败")
		return
	}
	web.Success(c, "")
	return
}

// UploadMergeHandler     合并分片文件
//
//	@Summary		合并分片文件
//	@Description	合并分片文件
//	@Tags			上传
//	@Accept			multipart/form-data
//	@Param			uid			query	string	true	"文件uid"
//	@Param			md5			query	string	true	"md5"
//	@Param			num			query	string	true	"总分片数量"
//	@Param			size		query	string	true	"文件总大小"
//	@Param			date		query	string	true	"链接生成时间"
//	@Param			expire		query	string	true	"过期时间"
//	@Param			signature	query	string	true	"签名"
//	@Produce		application/json
//	@Success		200	{object}	web.Response
//	@Router			/api/storage/v0/upload/merge [put]
func UploadMergeHandler(c *gin.Context) {
	uidStr := c.Query("uid")
	md5 := c.Query("md5")
	numStr := c.Query("num")
	size := c.Query("size")
	date := c.Query("date")
	expireStr := c.Query("expire")
	signature := c.Query("signature")

	uid, err, errorInfo := base.CheckValid(uidStr, date, expireStr)
	if err != nil {
		web.ParamsError(c, errorInfo)
		return
	}

	num, err := strconv.ParseInt(numStr, 10, 64)
	if err != nil {
		web.ParamsError(c, errorInfo)
		return
	}

	if !base.CheckUploadSignature(date, expireStr, signature) {
		web.ParamsError(c, "签名校验失败")
		return
	}

	// 判断记录是否存在
	lgDB := new(plugins.LangGoDB).Use("default").NewDB()
	metaData, err := repo.NewMetaDataInfoRepo().GetByUid(lgDB, uid)
	if err != nil {
		web.NotFoundResource(c, "当前合并链接无效，uid不存在")
		return
	}

	// 判断分片数量是否一致
	var multiPartInfoList []models.MultiPartInfo
	if err := lgDB.Model(&models.MultiPartInfo{}).Where(
		"storage_uid = ? and status = ?", uid, 1).Order("chunk_num ASC").Find(&multiPartInfoList).Error; err != nil {
		lgLogger.WithContext(c).Error("查询分片数据失败")
		web.InternalError(c, "查询分片数据失败")
		return
	}

	if num != int64(len(multiPartInfoList)) {
		// 创建脏数据删除任务
		msg := models.MergeInfo{
			StorageUid: uid,
			ChunkSum:   num,
		}
		b, err := json.Marshal(msg)
		if err != nil {
			lgLogger.WithContext(c).Error("消息struct转成json字符串失败", zap.Any("err", err.Error()))
			web.InternalError(c, "分片数量和整体数量不一致，创建删除任务失败")
			return
		}
		newModelTask := models.TaskInfo{
			Status:    utils.TaskStatusUndo,
			TaskType:  utils.TaskPartDelete,
			ExtraData: string(b),
		}
		if err := repo.NewTaskRepo().Create(lgDB, &newModelTask); err != nil {
			lgLogger.WithContext(c).Error("分片数量和整体数量不一致，创建删除任务失败", zap.Any("err", err.Error()))
			web.InternalError(c, "分片数量和整体数量不一致，创建删除任务失败")
			return
		}
		web.ParamsError(c, "分片数量和整体数量不一致")
		return
	}

	// 判断是否在本地
	dirName := path.Join(utils.LocalStore, uidStr)
	if _, err := os.Stat(dirName); os.IsNotExist(err) {
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
		_, _, _, err = thirdparty.NewStorageService().MergeForward(c, utils.Scheme, proxyIP,
			bootstrap.NewConfig("").App.Port, uidStr)
		if err != nil {
			lgLogger.WithContext(c).Error("合并文件，转发失败")
			web.InternalError(c, err.Error())
			return
		}
		web.Success(c, "")
		return
	}
	// 获取文件的content-type
	firstPart := multiPartInfoList[0]
	partName := path.Join(utils.LocalStore, fmt.Sprintf("%d", uid), firstPart.PartFileName)
	contentType, err := base.DetectContentType(partName)
	if err != nil {
		lgLogger.WithContext(c).Error("判断文件content-type失败")
		web.InternalError(c, "判断文件content-type失败")
		return
	}

	// 更新metadata的数据
	now := time.Now()
	if err := repo.NewMetaDataInfoRepo().Updates(lgDB, metaData.UID, map[string]interface{}{
		"part_num":     int(num),
		"md5":          md5,
		"storage_size": size,
		"multi_part":   true,
		"status":       1,
		"updated_at":   &now,
		"content_type": contentType,
	}); err != nil {
		lgLogger.WithContext(c).Error("上传完更新数据失败")
		web.InternalError(c, "上传完更新数据失败")
		return
	}
	// 创建合并任务
	msg := models.MergeInfo{
		StorageUid: uid,
		ChunkSum:   num,
	}
	b, err := json.Marshal(msg)
	if err != nil {
		lgLogger.WithContext(c).Error("消息struct转成json字符串失败", zap.Any("err", err.Error()))
		web.InternalError(c, "创建合并任务失败")
		return
	}
	newModelTask := models.TaskInfo{
		Status:    utils.TaskStatusUndo,
		TaskType:  utils.TaskPartMerge,
		ExtraData: string(b),
	}
	if err := repo.NewTaskRepo().Create(lgDB, &newModelTask); err != nil {
		lgLogger.WithContext(c).Error("创建合并任务失败", zap.Any("err", err.Error()))
		web.InternalError(c, "创建合并任务失败")
		return
	}

	// 首次写入redis 元数据和分片信息
	lgRedis := new(plugins.LangGoRedis).NewRedis()
	metaCache, err := repo.NewMetaDataInfoRepo().GetByUid(lgDB, uid)
	if err != nil {
		lgLogger.WithContext(c).Error("上传数据，查询数据元信息失败")
		web.InternalError(c, "内部异常")
		return
	}
	b, err = json.Marshal(metaCache)
	if err != nil {
		lgLogger.WithContext(c).Warn("上传数据，写入redis失败")
	}
	lgRedis.SetNX(context.Background(), fmt.Sprintf("%s-meta", uidStr), b, 5*60*time.Second)

	var multiPartInfoListCache []models.MultiPartInfo
	if err := lgDB.Model(&models.MultiPartInfo{}).Where(
		"storage_uid = ? and status = ?", uid, 1).Order("chunk_num ASC").Find(&multiPartInfoListCache).Error; err != nil {
		lgLogger.WithContext(c).Error("上传数据，查询分片数据失败")
		web.InternalError(c, "查询分片数据失败")
		return
	}
	// 写入redis
	b, err = json.Marshal(multiPartInfoListCache)
	if err != nil {
		lgLogger.WithContext(c).Warn("上传数据，写入redis失败")
	}
	lgRedis.SetNX(context.Background(), fmt.Sprintf("%s-multiPart", uidStr), b, 5*60*time.Second)

	web.Success(c, "")
	return
}
