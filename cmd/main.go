package main

import (
	"github.com/qinguoyi/ObjectStorageProxy/api"
	"github.com/qinguoyi/ObjectStorageProxy/app"
	"github.com/qinguoyi/ObjectStorageProxy/app/pkg/base"
	"github.com/qinguoyi/ObjectStorageProxy/app/pkg/storage"
	"github.com/qinguoyi/ObjectStorageProxy/bootstrap"
	"github.com/qinguoyi/ObjectStorageProxy/bootstrap/plugins"
)

// @title		ObjectStorageProxy
// @version	1.0
// @description
// @contact.name	qinguoyi
// @host			127.0.0.1:8888
// @BasePath		/
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
