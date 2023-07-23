package plugins

import (
	"context"
	"fmt"
	"github.com/qinguoyi/osproxy/bootstrap"
	"github.com/qinguoyi/osproxy/config"
	"github.com/tencentyun/cos-go-sdk-v5"
	"go.uber.org/zap"
	"net/http"
	"net/url"
	"sync"
)

var lgCos = new(LangGoCos)

type LangGoCos struct {
	Once      *sync.Once
	CosClient *cos.Client
}

func (lg *LangGoCos) NewCos() *cos.Client {
	if lgCos.CosClient != nil {
		return lgCos.CosClient
	} else {
		return lg.New().(*cos.Client)
	}
}

func newLangGoCos() *LangGoCos {
	return &LangGoCos{
		CosClient: &cos.Client{},
		Once:      &sync.Once{},
	}
}

func (lg *LangGoCos) Name() string {
	return "Cos"
}

func (lg *LangGoCos) New() interface{} {
	lgCos = newLangGoCos()
	lgCos.initCos(bootstrap.NewConfig(""))
	return lg.CosClient
}

func (lg *LangGoCos) Health() {
	ok, err := lgCos.CosClient.Bucket.IsExist(context.Background())
	if err == nil && ok {
		return
	} else if err != nil {
		bootstrap.NewLogger().Logger.Error("Cos connect failed, err:", zap.Any("err", err))
		panic("failed to connect cos")
	} else {
		return
	}
}

func (lg *LangGoCos) Close() {}

// Flag .
func (lg *LangGoCos) Flag() bool {
	return bootstrap.NewConfig("").Cos.Enabled
}

func init() {
	p := &LangGoCos{}
	RegisteredPlugin(p)
}

func (lg *LangGoCos) initCos(conf *config.Configuration) {
	lg.Once.Do(func() {
		appid := conf.Cos.Appid
		region := conf.Cos.Region
		secretId := conf.Cos.SecretId
		secretKey := conf.Cos.SecretKey
		u, _ := url.Parse(fmt.Sprintf("https://example-%s.cos.%s.myqcloud.com", appid, region))
		b := &cos.BaseURL{BucketURL: u}
		lgCos.CosClient = cos.NewClient(b, &http.Client{
			Transport: &cos.AuthorizationTransport{
				SecretID:  secretId,
				SecretKey: secretKey,
			},
		})
	})
}
