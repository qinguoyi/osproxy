package v0

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/qinguoyi/osproxy/app/pkg/base"
	"github.com/qinguoyi/osproxy/app/pkg/utils"
	"github.com/qinguoyi/osproxy/app/pkg/web"
	"os"
	"path"
	"strconv"
)

// IsOnCurrentServerHandler   .
//
//	@Summary		询问文件是否在当前服务
//	@Description	询问文件是否在当前服务
//	@Tags			proxy
//	@Accept			application/json
//	@Param			uid	query	string	true	"uid"
//	@Produce		application/json
//	@Success		200	{object}	web.Response
//	@Router			/api/storage/v0/proxy [get]
func IsOnCurrentServerHandler(c *gin.Context) {
	uidStr := c.Query("uid")
	_, err := strconv.ParseInt(uidStr, 10, 64)
	if err != nil {
		web.ParamsError(c, fmt.Sprintf("uid参数有误，详情:%s", err))
		return
	}
	dirName := path.Join(utils.LocalStore, uidStr)
	if _, err := os.Stat(dirName); os.IsNotExist(err) {
		web.NotFoundResource(c, "")
		return
	} else {
		ip, err := base.GetOutBoundIP()
		if err != nil {
			panic(err)
		}
		web.Success(c, ip)
		return
	}
}
