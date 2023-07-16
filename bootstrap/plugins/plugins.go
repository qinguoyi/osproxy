package plugins

import (
	"fmt"
	"github.com/qinguoyi/osproxy/bootstrap"
)

// Plugin 插件接口
type Plugin interface {
	// Flag 是否启动
	Flag() bool
	// Name 插件名称
	Name() string
	// New 初始化插件资源
	New() interface{}
	// Health 插件健康检查
	Health()
	// Close 释放插件资源
	Close()
}

// Plugins 插件注册集合
var Plugins = make(map[string]Plugin)

// RegisteredPlugin 插件注册
func RegisteredPlugin(plugin Plugin) {
	Plugins[plugin.Name()] = plugin
}

func NewPlugins() {
	for _, p := range Plugins {
		if !p.Flag() {
			continue
		}
		bootstrap.NewLogger().Logger.Info(fmt.Sprintf("%s Init ... ", p.Name()))
		p.New()
		bootstrap.NewLogger().Logger.Info(fmt.Sprintf("%s HealthCheck ... ", p.Name()))
		p.Health()
		bootstrap.NewLogger().Logger.Info(fmt.Sprintf("%s Success Init. ", p.Name()))
	}
}

func ClosePlugins() {
	for _, p := range Plugins {
		if !p.Flag() {
			continue
		}
		p.Close()
		bootstrap.NewLogger().Logger.Info(fmt.Sprintf("%s Success Close ... ", p.Name()))
	}
}
