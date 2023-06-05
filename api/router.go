package api

import (
	"github.com/gin-gonic/gin"
	v0 "github.com/qinguoyi/ObjectStorageProxy/api/v0"
	"github.com/qinguoyi/ObjectStorageProxy/app/middleware"
	"github.com/qinguoyi/ObjectStorageProxy/bootstrap"
	"github.com/qinguoyi/ObjectStorageProxy/config"
	"github.com/qinguoyi/ObjectStorageProxy/docs"
	gs "github.com/swaggo/gin-swagger"
	"github.com/swaggo/gin-swagger/swaggerFiles"
)

func NewRouter(
	conf *config.Configuration,
	lgLogger *bootstrap.LangGoLogger,
) *gin.Engine {
	if conf.App.Env == "prod" {
		gin.SetMode(gin.ReleaseMode)
	}
	router := gin.New()

	// middleware
	corsM := middleware.NewCors()
	traceL := middleware.NewTrace(lgLogger)
	requestL := middleware.NewRequestLog(lgLogger)
	panicRecover := middleware.NewPanicRecover(lgLogger)

	// 跨域 trace-id 日志
	router.Use(corsM.Handler(), traceL.Handler(), requestL.Handler(), panicRecover.Handler())

	// 静态资源
	router.StaticFile("/assets", "../../static/image/back.png")

	// swag docs
	docs.SwaggerInfo.BasePath = "/"
	router.GET("/swagger/*any", gs.WrapHandler(swaggerFiles.Handler))

	// 动态资源 注册 api 分组路由
	setApiGroupRoutes(router)

	return router
}

func setApiGroupRoutes(
	router *gin.Engine,
) *gin.RouterGroup {
	group := router.Group("/api/storage/v0")
	{
		//health
		group.GET("/ping", v0.PingHandler)
		group.GET("/health", v0.HealthCheckHandler)

		// resume
		group.POST("/resume", v0.ResumeHandler)
		group.GET("/checkpoint", v0.CheckPointHandler)

		// link
		group.POST("/link/upload", v0.UploadLinkHandler)
		group.POST("/link/download", v0.DownloadLinkHandler)

		// proxy
		group.GET("/proxy", v0.IsOnCurrentServerHandler)

		// upload
		group.PUT("/upload", v0.UploadSingleHandler)
		group.PUT("/upload/multi", v0.UploadMultiPartHandler)
		group.PUT("/upload/merge", v0.UploadMergeHandler)

		//download
		group.GET("/download", v0.DownloadHandler)

	}
	return group
}
