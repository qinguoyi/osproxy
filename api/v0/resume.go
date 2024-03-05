package v0

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/qinguoyi/osproxy/app/models"
	"github.com/qinguoyi/osproxy/app/pkg/base"
	"github.com/qinguoyi/osproxy/app/pkg/repo"
	"github.com/qinguoyi/osproxy/app/pkg/utils"
	"github.com/qinguoyi/osproxy/app/pkg/web"
	"github.com/qinguoyi/osproxy/bootstrap/plugins"
	"go.uber.org/zap"
	"path/filepath"
	"time"
)

/*
秒传及断点续传(上传)
*/

// ResumeHandler    秒传&断点续传
//
//	@Summary      秒传&断点续传
//	@Description  秒传&断点续传
//	@Tags         秒传
//	@Accept       application/json
//	@Param        RequestBody  body  models.ResumeReq  true  "秒传请求体"
//	@Produce      application/json
//	@Success      200  {object}  web.Response{data=[]models.ResumeResp}
//	@Router       /api/storage/v0/resume [post]
func ResumeHandler(c *gin.Context) {
	resumeReq := models.ResumeReq{}
	if err := c.ShouldBindJSON(&resumeReq); err != nil {
		web.ParamsError(c, "参数有误")
		return
	}
	if len(resumeReq.Data) > utils.LinkLimit {
		web.ParamsError(c, fmt.Sprintf("判断文件秒传，数量不能超过%d个", utils.LinkLimit))
		return
	}

	var md5List []string
	md5MapName := map[string]string{}
	for _, i := range resumeReq.Data {
		md5MapName[i.Md5] = i.Path
		md5List = append(md5List, i.Md5)
	}
	md5List = utils.RemoveDuplicates(md5List)

	md5MapResp := map[string]*models.ResumeResp{}
	for _, md5 := range md5List {
		tmp := models.ResumeResp{
			Uid: "",
			Md5: md5,
		}
		md5MapResp[md5] = &tmp
	}

	// 秒传只看已上传且完整文件的数据
	lgDB := new(plugins.LangGoDB).Use("default").NewDB()
	resumeInfo, err := repo.NewMetaDataInfoRepo().GetResumeByMd5(lgDB, md5List)
	if err != nil {
		lgLogger.WithContext(c).Error("查询秒传数据失败")
		web.InternalError(c, "")
		return
	}
	// 去重
	md5MapMetaInfo := map[string]models.MetaDataInfo{}
	for _, resume := range resumeInfo {
		if _, ok := md5MapMetaInfo[resume.Md5]; !ok {
			md5MapMetaInfo[resume.Md5] = resume
		}
	}

	var newMetaDataList []models.MetaDataInfo
	for _, resume := range resumeReq.Data {
		if _, ok := md5MapMetaInfo[resume.Md5]; !ok {
			continue
		}
		// 相同数据上传需要复制一份数据
		uid, _ := base.NewSnowFlake().NextId()
		now := time.Now()
		newMetaDataList = append(newMetaDataList,
			models.MetaDataInfo{
				UID:         uid,
				Bucket:      md5MapMetaInfo[resume.Md5].Bucket,
				Name:        filepath.Base(resume.Path),
				StorageName: md5MapMetaInfo[resume.Md5].StorageName,
				Address:     md5MapMetaInfo[resume.Md5].Address,
				Md5:         resume.Md5,
				MultiPart:   false,
				StorageSize: md5MapMetaInfo[resume.Md5].StorageSize,
				Status:      1,
				ContentType: md5MapMetaInfo[resume.Md5].ContentType,
				CreatedAt:   &now,
				UpdatedAt:   &now,
			})
		md5MapResp[resume.Md5].Uid = fmt.Sprintf("%d", uid)
	}
	if len(newMetaDataList) != 0 {
		if err := repo.NewMetaDataInfoRepo().BatchCreate(lgDB, &newMetaDataList); err != nil {
			lgLogger.WithContext(c).Error("秒传批量落数据库失败，详情：", zap.Any("err", err.Error()))
			web.InternalError(c, "内部异常")
			return
		}
	}
	lgRedis := new(plugins.LangGoRedis).NewRedis()
	for _, metaDataCache := range newMetaDataList {
		b, err := json.Marshal(metaDataCache)
		if err != nil {
			lgLogger.WithContext(c).Warn("秒传数据，写入redis失败")
		}
		lgRedis.SetNX(context.Background(), fmt.Sprintf("%d-meta", metaDataCache.UID), b, 5*60*time.Second)
	}

	var respList []models.ResumeResp
	for _, resp := range md5MapResp {
		respList = append(respList, *resp)
	}
	web.Success(c, respList)
	return
}
