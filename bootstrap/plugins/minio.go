package plugins

import (
	"context"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/qinguoyi/osproxy/bootstrap"
	"github.com/qinguoyi/osproxy/config"
	"go.uber.org/zap"
	"sync"
)

var lgMinio = new(LangGoMinio)

type LangGoMinio struct {
	Once        *sync.Once
	MinioClient *minio.Client
}

func (lg *LangGoMinio) NewMinio() *minio.Client {
	if lgMinio.MinioClient != nil {
		return lgMinio.MinioClient
	} else {
		return lg.New().(*minio.Client)
	}
}

func newLangGoMinio() *LangGoMinio {
	return &LangGoMinio{
		MinioClient: &minio.Client{},
		Once:        &sync.Once{},
	}
}

func (lg *LangGoMinio) Name() string {
	return "Minio"
}

func (lg *LangGoMinio) New() interface{} {
	lgMinio = newLangGoMinio()
	lgMinio.initMinio(bootstrap.NewConfig(""))
	return lg.MinioClient
}

func (lg *LangGoMinio) Health() {
	_, err := lgMinio.MinioClient.ListBuckets(context.Background())
	if err != nil {
		bootstrap.NewLogger().Logger.Error("Minio connect failed, err:", zap.Any("err", err))
		panic("failed to connect minio")
	}
}

func (lg *LangGoMinio) Close() {}

// Flag .
func (lg *LangGoMinio) Flag() bool { return bootstrap.NewConfig("").Minio.Enabled }

func init() {
	p := &LangGoMinio{}
	RegisteredPlugin(p)
}

func (lg *LangGoMinio) initMinio(conf *config.Configuration) {
	lg.Once.Do(func() {
		endpoint := conf.Minio.EndPoint
		accessKeyID := conf.Minio.AccessKeyID
		secretAccessKey := conf.Minio.SecretAccessKey
		useSSL := conf.Minio.UseSSL
		client, err := minio.New(endpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
			Secure: useSSL,
		})

		if err != nil {
			bootstrap.NewLogger().Logger.Error("minio连接错误: ", zap.Any("err", err))
			panic(err)
		} else {
			lgMinio.MinioClient = client
		}
	})
}
