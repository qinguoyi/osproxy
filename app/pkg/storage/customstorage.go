package storage

import (
	"github.com/qinguoyi/ObjectStorageProxy/bootstrap"
	"github.com/qinguoyi/ObjectStorageProxy/config"
	"sync"
)

// CustomStorage 存储
type CustomStorage interface {
	// MakeBucket 创建存储桶
	MakeBucket(string) error

	// GetObject 获取存储对象
	GetObject(string, string, int64, int64) ([]byte, error)

	// PutObject 上传存储对象
	PutObject(string, string, string, string) error

	// DeleteObject 删除存储对象
	DeleteObject(string, string) error
}

type LangGoStorage struct {
	Mux     *sync.RWMutex
	Storage CustomStorage
}

var (
	lgStorage *LangGoStorage
)

func InitStorage(conf *config.Configuration) {
	var storageHandler CustomStorage
	if conf.Local.Enabled {
		storageHandler = NewLocalStorage()
		bootstrap.NewLogger().Logger.Info("当前使用的对象存储：Local")
	} else if conf.Minio.Enabled {
		storageHandler = NewMinIOStorage()
		bootstrap.NewLogger().Logger.Info("当前使用的对象存储：Minio")
	} else if conf.Cos.Enabled {
		storageHandler = NewCosStorage()
		bootstrap.NewLogger().Logger.Info("当前使用的对象存储：COS")
	} else if conf.Oss.Enabled {
		storageHandler = NewOssStorage()
		bootstrap.NewLogger().Logger.Info("当前使用的对象存储：OSS")
	} else {
		panic("当前对象存储都未启用")
	}

	lgStorage = &LangGoStorage{
		Mux:     &sync.RWMutex{},
		Storage: storageHandler,
	}
	for _, bucket := range []string{"image", "video", "audio", "archive", "unknown", "doc"} {
		if err := storageHandler.MakeBucket(bucket); err != nil {
			panic(err)
		}
	}
}

func NewStorage() *LangGoStorage {
	if lgStorage != nil {
		return lgStorage
	} else {
		return nil
	}
}
