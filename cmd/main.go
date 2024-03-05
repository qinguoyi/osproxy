package main

import (
	"github.com/qinguoyi/osproxy/api"
	"github.com/qinguoyi/osproxy/app"
	"github.com/qinguoyi/osproxy/app/pkg/base"
	"github.com/qinguoyi/osproxy/app/pkg/storage"
	"github.com/qinguoyi/osproxy/bootstrap"
	"github.com/qinguoyi/osproxy/bootstrap/plugins"
)

// @title    ObjectStorageProxy
// @version  1.0
// @description
// @contact.name  qinguoyi
// @host          127.0.0.1:8888
// @BasePath      /
func main() {
	// config log
	lgConfig := bootstrap.NewConfig("conf/config.yaml")
	lgLogger := bootstrap.NewLogger()

	// plugins DB Redis Minio
	plugins.NewPlugins()
	defer plugins.ClosePlugins()

	// init Snowflake
	base.InitSnowFlake()

	// init storage
	storage.InitStorage(lgConfig)

	// router
	engine := api.NewRouter(lgConfig, lgLogger)
	server := app.NewHttpServer(lgConfig, engine)

	// app run-server
	application := app.NewApp(lgConfig, lgLogger.Logger, server)
	application.RunServer()
}
