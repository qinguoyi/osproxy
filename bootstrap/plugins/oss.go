package plugins

import (
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/qinguoyi/ObjectStorageProxy/bootstrap"
	"github.com/qinguoyi/ObjectStorageProxy/config"
	"go.uber.org/zap"
	"sync"
)

var lgOss = new(LangGoOss)

type LangGoOss struct {
	Once      *sync.Once
	OssClient *oss.Client
}

func (lg *LangGoOss) NewOss() *oss.Client {
	if lgOss.OssClient != nil {
		return lgOss.OssClient
	} else {
		return lg.New().(*oss.Client)
	}
}

func newLangGoOss() *LangGoOss {
	return &LangGoOss{
		OssClient: &oss.Client{},
		Once:      &sync.Once{},
	}
}

func (lg *LangGoOss) Name() string {
	return "Oss"
}

func (lg *LangGoOss) New() interface{} {
	lgOss = newLangGoOss()
	lgOss.initOss(bootstrap.NewConfig(""))
	return lg.OssClient
}

func (lg *LangGoOss) Health() {
	_, err := lgOss.OssClient.IsBucketExist("example")
	if err != nil {
		bootstrap.NewLogger().Logger.Error("oss connect failed, err:", zap.Any("err", err))
		panic("failed to connect oss")
	} else {
		return
	}
}

func (lg *LangGoOss) Close() {}

// Flag .
func (lg *LangGoOss) Flag() bool {
	return bootstrap.NewConfig("").Oss.Enabled
}

func init() {
	p := &LangGoOss{}
	RegisteredPlugin(p)
}

func (lg *LangGoOss) initOss(conf *config.Configuration) {
	lg.Once.Do(func() {
		endpoint := conf.Oss.EndPoint
		accessKeyId := conf.Oss.AccessKeyId
		accessKeySecret := conf.Oss.AccessKeySecret
		client, err := oss.New(endpoint, accessKeyId, accessKeySecret)
		if err != nil {
			bootstrap.NewLogger().Logger.Error("oss 连接失败, err:", zap.Any("err", err))
			panic(err)
		}
		lgOss.OssClient = client
	})
}
