package plugins

import (
	"context"
	"github.com/go-redis/redis/extra/redisotel"
	"github.com/go-redis/redis/v8"
	"github.com/qinguoyi/osproxy/bootstrap"
	"github.com/qinguoyi/osproxy/config"
	"go.uber.org/zap"
	"sync"
)

var lgRedis = new(LangGoRedis)

type LangGoRedis struct {
	Once        *sync.Once
	RedisClient *redis.Client
}

func (lg *LangGoRedis) NewRedis() *redis.Client {
	if lgRedis.RedisClient != nil {
		return lgRedis.RedisClient
	} else {
		return lg.New().(*redis.Client)
	}
}

func newLangGoRedis() *LangGoRedis {
	return &LangGoRedis{
		RedisClient: &redis.Client{},
		Once:        &sync.Once{},
	}
}

func (lg *LangGoRedis) Name() string {
	return "Redis"
}

func (lg *LangGoRedis) New() interface{} {
	lgRedis = newLangGoRedis()
	lgRedis.initRedis(bootstrap.NewConfig(""))
	return lgRedis.RedisClient
}

func (lg *LangGoRedis) Health() {
	if err := lgRedis.RedisClient.Ping(context.Background()).Err(); err != nil {
		bootstrap.NewLogger().Logger.Error("redis connect failed, err:", zap.Any("err", err))
		panic(err)
	}
}

func (lg *LangGoRedis) Close() {
	if lg.RedisClient == nil {
		return
	} else {
		if err := lg.RedisClient.Close(); err != nil {
			bootstrap.NewLogger().Logger.Error("redis close failed, err:", zap.Any("err", err))
		}
	}
}

// Flag .
func (lg *LangGoRedis) Flag() bool { return true }

func init() {
	p := &LangGoRedis{}
	RegisteredPlugin(p)
}

func (lg *LangGoRedis) initRedis(conf *config.Configuration) {
	lg.Once.Do(func() {
		client := redis.NewClient(&redis.Options{
			Addr:     conf.Redis.Host + ":" + conf.Redis.Port,
			Password: conf.Redis.Password, // no password set
			DB:       conf.Redis.DB,       // use default DB
		})

		// redis链路追踪相关
		client.AddHook(redisotel.TracingHook{})
		lgRedis.RedisClient = client
	})

}
