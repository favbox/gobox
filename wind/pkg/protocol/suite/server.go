package suite

import (
	"context"
	"sync"

	"github.com/favbox/gosky/wind/pkg/app"
	"github.com/favbox/gosky/wind/pkg/common/errors"
	"github.com/favbox/gosky/wind/pkg/common/hlog"
	"github.com/favbox/gosky/wind/pkg/common/tracer"
	"github.com/favbox/gosky/wind/pkg/protocol"
)

// Core 是承诺为协议层提供扩展的核心接口。
type Core interface {
	// IsRunning 检查引擎是否在运行。
	IsRunning() bool

	// GetCtxPool 用于实现协议服务器的上下文池
	GetCtxPool() *sync.Pool

	// ServeHTTP 业务逻辑入口。
	// 在预读完成后，协议服务器调此法引入中间件和处理器。
	ServeHTTP(c context.Context, ctx *app.RequestContext)

	// GetTracer 获取链路跟踪控制器。
	GetTracer() tracer.Controller
}

// ServerFactory 接口定义了创建普通服务器的工厂方法。
type ServerFactory interface {
	New(core Core) (server protocol.Server, err error)
}

// StreamServerFactory 接口定义了创建流式服务器的工厂方法。
type StreamServerFactory interface {
	New(core Core) (server protocol.StreamServer, err error)
}

type ServerMap map[string]protocol.Server

type StreamServerMap map[string]protocol.StreamServer

type altServerConfig struct {
	targetProtocol   string
	setAltHeaderFunc func(ctx context.Context, reqCtx *app.RequestContext)
}

type coreWrapper struct {
	Core
	beforeHandler func(c context.Context, ctx *app.RequestContext)
}

func (w *coreWrapper) ServeHTTP(c context.Context, ctx *app.RequestContext) {
	w.beforeHandler(c, ctx)
	w.Core.ServeHTTP(c, ctx)
}

// Config 用来维护协议与对应服务器的映射。
type Config struct {
	altServerConfig *altServerConfig               // 协议替补的服务器配置
	configMap       map[string]ServerFactory       // 协议对应的普通服务器工厂
	streamConfigMap map[string]StreamServerFactory // 协议对应的流式服务器工厂
}

// Add 添加给定协议的服务器工厂方法。
func (c *Config) Add(protocol string, factory any) {
	switch newFac := factory.(type) {
	case ServerFactory:
		if oldFac := c.configMap[protocol]; oldFac != nil {
			hlog.SystemLogger().Warnf("协议 %s 的服务器工厂将被自定义函数覆盖")
		}
		c.configMap[protocol] = newFac
	case StreamServerFactory:
		if oldFac := c.streamConfigMap[protocol]; oldFac != nil {
			hlog.SystemLogger().Warnf("协议 %s 的服务器工厂将被自定义函数覆盖")
		}
		c.streamConfigMap[protocol] = newFac
	default:
		hlog.SystemLogger().Fatalf("不支持的服务器工厂类型：%T", newFac)
	}
}

// Get 返回指定协议名称的普通服务器工厂。
func (c *Config) Get(protocol string) ServerFactory {
	return c.configMap[protocol]
}

// Delete 删除给定协议名称的普通服务器工厂。
func (c *Config) Delete(protocol string) {
	delete(c.configMap, protocol)
}

// Load 加载给定协议对应的普通服务器。
func (c *Config) Load(core Core, protocol string) (server protocol.Server, err error) {
	if c.configMap[protocol] == nil {
		return nil, errors.NewPrivate("WIND: 加载服务器出错，不支持的协议：" + protocol)
	}
	// 未指定替代协议的服务器，或给定的协议正好替代协议一致，则返回基于给定内核 core 创建的服务器。
	if c.altServerConfig == nil || c.altServerConfig.targetProtocol == protocol {
		return c.configMap[protocol].New(core)
	}
	// 否则，返回基于给定内核 core 包装后创建的服务器。
	return c.configMap[protocol].New(&coreWrapper{
		Core:          core,
		beforeHandler: c.altServerConfig.setAltHeaderFunc,
	})
}

// LoadAll 创建所有协议的服务器并返回映射。
func (c *Config) LoadAll(core Core) (serverMap ServerMap, streamServerMap StreamServerMap, err error) {
	// 预备一个包装后的内核
	var wrappedCore *coreWrapper
	if c.altServerConfig != nil {
		wrappedCore = &coreWrapper{
			Core:          core,
			beforeHandler: c.altServerConfig.setAltHeaderFunc,
		}
	}

	// 创建普通服务器并加入映射
	serverMap = make(ServerMap)
	var server protocol.Server
	for proto := range c.configMap {
		if c.altServerConfig != nil && c.altServerConfig.targetProtocol != proto {
			core = wrappedCore
		}
		if server, err = c.configMap[proto].New(core); err != nil {
			return nil, nil, err
		} else {
			serverMap[proto] = server
		}
	}

	// 创建流式服务器并加入映射
	streamServerMap = make(StreamServerMap)
	var streamServer protocol.StreamServer
	for proto := range c.streamConfigMap {
		if c.altServerConfig != nil && c.altServerConfig.targetProtocol != proto {
			core = wrappedCore
		}
		if streamServer, err = c.streamConfigMap[proto].New(core); err != nil {
			return nil, nil, err
		} else {
			streamServerMap[proto] = streamServer
		}
	}

	// 返回创建的协议与服务器映射
	return serverMap, streamServerMap, nil
}

// New 返回空白协议组配置，用 *Config.Add 来添加协议对应的服务器实现。
func New() *Config {
	c := &Config{
		configMap:       make(map[string]ServerFactory),
		streamConfigMap: make(map[string]StreamServerFactory),
	}
	return c
}
