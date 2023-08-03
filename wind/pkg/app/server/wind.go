package server

import (
	"github.com/favbox/gosky/wind/pkg/common/config"
	"github.com/favbox/gosky/wind/pkg/route"
)

// Wind 是 wind 的核心结构体。
//
// 组合了路由引擎 route.Engine 和 优雅退出函数。
type Wind struct {
	*route.Engine
	// 用于接收信息实现优雅退出
	signalWaiter func(err chan error) error
}

// New 创建一个无默认配置的 wind 实例。
func New(opts ...config.Option) *Wind {
	options := config.NewOptions(opts)
	w := &Wind{
		Engine: route.NewEngine(options),
	}
	return w
}
