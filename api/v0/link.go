package v0

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/qinguoyi/ObjectStorageProxy/app/models"
	"github.com/qinguoyi/ObjectStorageProxy/app/pkg/base"
	"github.com/qinguoyi/ObjectStorageProxy/app/pkg/repo"
	"github.com/qinguoyi/ObjectStorageProxy/app/pkg/utils"
	"github.com/qinguoyi/ObjectStorageProxy/app/pkg/web"
	"github.com/qinguoyi/ObjectStorageProxy/bootstrap/plugins"
	"go.uber.org/zap"
	"os"
	"path"
	"strconv"
	"sync"
)

/*
对象信息，生成连接(上传、下载)
*/

// UploadLinkHandler    初始化上传连接
//
//	@Summary		初始化上传连接
//	@Description	初始化上传连接
//	@Tags			链接
//	@Accept			application/json
//	@Param			RequestBody	body	models.GenUpload	true	"生成上传链接请求体"
//	@Produce		application/json
//	@Success		200	{object}	web.Response{data=models.GenUploadResp}
//	@Router			/api/storage/v0/link/upload [post]
func UploadLinkHandler(c *gin.Context) {
	var genUploadReq models.GenUpload
	if err := c.ShouldBindJSON(&genUploadReq); err != nil {
		web.ParamsError(c, fmt.Sprintf("参数解析有误，详情：%s", err))
		return
	}
	if len(genUploadReq.FilePath) > utils.LinkLimit {
		web.ParamsError(c, fmt.Sprintf("批量上传路径数量有限，最多%d条", utils.LinkLimit))
		return
	}

	// deduplication filepath
	fileNameList := utils.RemoveDuplicates(genUploadReq.FilePath)
	for _, fileName := range fileNameList {
		if base.GetExtension(fileName) == "" {
			web.ParamsError(c, fmt.Sprintf("文件[%s]后缀有误，不能为空", fileName))
			return
		}
	}

	var resp []models.GenUploadResp
	var resourceInfo []models.MetaDataInfo
	respChan := make(chan models.GenUploadResp, len(fileNameList))
	metaDataInfoChan := make(chan models.MetaDataInfo, len(fileNameList))

	var wg sync.WaitGroup
	for _, fileName := range fileNameList {
		wg.Add(1)
		go base.GenUploadSingle(fileName, genUploadReq.Expire, respChan, metaDataInfoChan, &wg)
	}
	wg.Wait()
	close(respChan)
	close(metaDataInfoChan)

	for re := range respChan {
		resp = append(resp, re)
	}
	for re := range metaDataInfoChan {
		resourceInfo = append(resourceInfo, re)
	}
	if !(len(resp) == len(resourceInfo) && len(resp) == len(fileNameList)) {
		// clean local dir
		for _, i := range resp {
			dirName := path.Join(utils.LocalStore, i.Uid)
			go func() {
				_ = os.RemoveAll(dirName)
			}()
		}
		lgLogger.WithContext(c).Error("生成链接，生成的url和输入数量不一致")
		web.InternalError(c, "内部异常")
		return
	}

	// db batch create
	lgDB := new(plugins.LangGoDB).Use("default").NewDB()
	if err := repo.NewMetaDataInfoRepo().BatchCreate(lgDB, &resourceInfo); err != nil {
		lgLogger.WithContext(c).Error("生成链接，批量落数据库失败，详情：", zap.Any("err", err.Error()))
		web.InternalError(c, "内部异常")
		return
	}
	web.Success(c, resp)
}

// DownloadLinkHandler    获取下载连接
//
//	@Summary		获取下载连接
//	@Description	获取下载连接
//	@Tags			链接
//	@Accept			application/json
//	@Param			RequestBody	body	models.GenDownload	true	"下载链接请求体"
//	@Produce		application/json
//	@Success		200	{object}	web.Response{data=models.GenDownloadResp}
//	@Router			/api/storage/v0/link/download [post]
func DownloadLinkHandler(c *gin.Context) {
	var genDownloadReq models.GenDownload
	if err := c.ShouldBindJSON(&genDownloadReq); err != nil {
		web.ParamsError(c, fmt.Sprintf("参数解析有误，详情：%s", err))
		return
	}

	if len(genDownloadReq.Uid) > 200 {
		web.ParamsError(c, "uid获取下载链接，数量不能超过200个")
		return
	}
	expireStr := fmt.Sprintf("%d", genDownloadReq.Expire)
	var uidList []int64
	var resp []models.GenDownloadResp
	for _, uidStr := range utils.RemoveDuplicates(genDownloadReq.Uid) {
		uid, err := strconv.ParseInt(uidStr, 10, 64)
		if err != nil {
			web.ParamsError(c, "uid参数有误")
			return
		}

		// 查询redis
		key := fmt.Sprintf("%d-%s", uid, expireStr)
		lgRedis := new(plugins.LangGoRedis).NewRedis()
		val, err := lgRedis.Get(context.Background(), key).Result()
		// key在redis中不存在
		if err == redis.Nil {
			uidList = append(uidList, uid)
			continue
		}
		if err != nil {
			lgLogger.WithContext(c).Error("获取下载链接，查询redis失败")
			web.InternalError(c, "")
			return
		}
		var msg models.GenDownloadResp
		if err := json.Unmarshal([]byte(val), &msg); err != nil {
			lgLogger.WithContext(c).Error("查询redis结果，序列化失败")
			web.InternalError(c, "")
			return
		}
		resp = append(resp, msg)
	}

	lgDB := new(plugins.LangGoDB).Use("default").NewDB()
	metaList, err := repo.NewMetaDataInfoRepo().GetByUidList(lgDB, uidList)
	if err != nil {
		lgLogger.WithContext(c).Error("获取下载链接，查询元数据信息失败")
		web.InternalError(c, "内部异常")
		return
	}
	if len(metaList) == 0 && len(resp) == 0 {
		web.Success(c, nil)
		return
	}
	uidMapMeta := map[int64]models.MetaDataInfo{}
	for _, meta := range metaList {
		uidMapMeta[meta.UID] = meta
	}

	respChan := make(chan models.GenDownloadResp, len(metaList))
	var wg sync.WaitGroup
	for _, uid := range uidList {
		wg.Add(1)
		go base.GenDownloadSingle(uidMapMeta[uid], expireStr, respChan, &wg)
	}
	wg.Wait()
	close(respChan)

	for re := range respChan {
		resp = append(resp, re)
	}
	web.Success(c, resp)
	return
}
