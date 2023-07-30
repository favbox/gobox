package suite

import (
	"context"
	"sync"

	"github.com/favbox/gosky/wind/pkg/app"
	"github.com/favbox/gosky/wind/pkg/common/tracer"
	"github.com/favbox/gosky/wind/pkg/protocol"
)

// Core 是承诺为协议层扩展提供的核心接口。
type Core interface {
	// IsRunning 检查引擎是否在运行。
	IsRunning() bool

	// GetCtxPool 用于实现协议服务器的上下文池
	GetCtxPool() *sync.Pool

	// ServeHTTP 业务逻辑入库。
	// 在预读完成后，协议服务器调此方法来引入中间件和处理器。
	ServeHTTP(c context.Context, ctx *app.RequestContext)

	// GetTracer 获取链路跟踪控制器。
	GetTracer() tracer.Controller
}

type ServerFactory interface {
	New(core Core) (protocol.Server, error)
}

type Config struct {
	altServerConfig *altServerConfig
	configMap       map[string]ServerFactory
}

type altServerConfig struct {
	targetProtocol   string
	setAltHeaderFunc func(ctx context.Context, reqCtx *app.RequestContext)
}
