package v0

import (
	"github.com/gin-gonic/gin"
	"github.com/qinguoyi/ObjectStorageProxy/app/pkg/repo"
	"github.com/qinguoyi/ObjectStorageProxy/app/pkg/web"
	"github.com/qinguoyi/ObjectStorageProxy/bootstrap/plugins"
	"strconv"
)

// CheckPointHandler    断点续传
//
//	@Summary		断点续传
//	@Description	断点续传
//	@Tags			断点续传
//	@Accept			application/json
//	@Param			uid	query	string	true	"文件uid"
//	@Produce		application/json
//	@Success		200	{object}	web.Response{data=[]int}
//	@Router			/api/storage/v0/checkpoint [get]
func CheckPointHandler(c *gin.Context) {
	uidStr := c.Query("uid")
	uid, err := strconv.ParseInt(uidStr, 10, 64)
	if err != nil {
		web.ParamsError(c, "uid参数有误")
		return
	}

	// 断点续传只看未上传且分片的数据
	lgDB := new(plugins.LangGoDB).Use("default").NewDB()
	partUidInfo, err := repo.NewMetaDataInfoRepo().GetPartByUid(lgDB, uid)
	if err != nil {
		lgLogger.WithContext(c).Error("查询断点续传数据失败")
		web.InternalError(c, "")
		return
	}
	if len(partUidInfo) == 0 {
		web.ParamsError(c, "当前文件uid不存在分片数据")
		return
	}

	// 断点续传查询分片数字
	partNumInfo, err := repo.NewMultiPartInfoRepo().GetPartNumByUid(lgDB, uid)
	var partNum []int
	for _, partInfo := range partNumInfo {
		partNum = append(partNum, partInfo.ChunkNum)
	}
	web.Success(c, partNum)
	return
}
