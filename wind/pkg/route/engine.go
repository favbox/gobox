package route

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/favbox/gosky/wind/internal/nocopy"
	"github.com/favbox/gosky/wind/pkg/app"
	"github.com/favbox/gosky/wind/pkg/common/config"
	errs "github.com/favbox/gosky/wind/pkg/common/errors"
	"github.com/favbox/gosky/wind/pkg/common/hlog"
	"github.com/favbox/gosky/wind/pkg/common/tracer"
	"github.com/favbox/gosky/wind/pkg/common/tracer/stats"
	"github.com/favbox/gosky/wind/pkg/common/tracer/traceinfo"
	"github.com/favbox/gosky/wind/pkg/common/utils"
	"github.com/favbox/gosky/wind/pkg/network"
	"github.com/favbox/gosky/wind/pkg/network/standard"
	"github.com/favbox/gosky/wind/pkg/protocol"
	"github.com/favbox/gosky/wind/pkg/protocol/suite"
)

const unknownTransporterName = "unknown"

var (
	defaultTransporter = standard.NewTransporter

	errInitFailed       = errs.NewPrivate("路由引擎已被初始化")
	errAlreadyRunning   = errs.NewPrivate("路由引擎已在运行中")
	errStatusNotRunning = errs.NewPrivate("路由引擎未在运行中")

	default404Body = []byte("404 页面未找到")
	default405Body = []byte("405 方法不允许")
	default400Body = []byte("400 错误请求")
)

// CtxCallback 引擎启动时，依次触发的钩子函数
type CtxCallback func(ctx context.Context)

// CtxErrCallback 引擎关闭时，同时触发的钩子函数
type CtxErrCallback func(ctx context.Context) error

type Engine struct {
	noCopy nocopy.NoCopy

	// 引擎名称
	Name       string
	serverName atomic.Value

	// 路由和协议服务器的配置项
	options *config.Options

	// 路由
	RouterGroup
	trees MethodTrees

	// 最大路由参数个数
	maxParams uint16

	allNoMethod app.HandlersChain
	allNoRoute  app.HandlersChain
	noRoute     app.HandlersChain
	noMethod    app.HandlersChain

	// TODO 用于渲染 HTML

	// 底层传输的网络库，现有 go net 和 netpoll l两个选择
	transport network.Transporter

	// 用于链路追踪
	tracerCtl   tracer.Controller
	enableTrace bool

	// TODO 用于管理协议层
	protocolSuite *suite.Config

	// RequestContext 连接池
	ctxPool sync.Pool

	// 处理从 http 处理器中恢复的 panic 的函数。
	// 用于生成错误页并返回 http 错误代码 500（内部服务器错误）。
	// 该处理器可防止服务器因未回复的 panic 而崩溃。
	PanicHandler app.HandlerFunc

	// 在收到 Expect 100 Continue 标头后调用 ContinueHandler。
	// 使用该处理器，服务器可以基于头信息决定是否读取可能较大的请求正文。
	//
	// 默认会自动读取请求体，就像普通请求一样。
	ContinueHandler func(header *protocol.RequestHeader) bool

	// 用于表示引擎状态（Init/Running/Shutdown/CLosed）
	status uint32

	// 引擎启动时，依次触发的钩子函数
	OnRun []CtxErrCallback

	// 引擎关闭时，同时触发的钩子函数
	OnShutdown []CtxCallback

	// 自定义函数
	clientIPFunc  app.ClientIP
	formValueFunc app.FormValueFunc
}

func (engine *Engine) AddProtocol(protocol string, factory any) {
	engine.protocolSuite.Add(protocol, factory)
}

// NewContext 创建一个无请求/响应的纯粹请求上下文。
func (engine *Engine) NewContext() *app.RequestContext {
	return app.NewContext(engine.maxParams)
}

func (engine *Engine) SetClientIPFunc(f app.ClientIP) {
	engine.clientIPFunc = f
}

func (engine *Engine) addRoute(method, path string, handlers app.HandlersChain) {
	if len(path) == 0 {
		panic("路径不能为空")
	}
	utils.Assert(path[0] == '/', "路径必须以 / 开头")
	utils.Assert(method != "", "HTTP 方法不能为空")
	utils.Assert(len(handlers) > 0, "至少要对应一个处理器")

	if !engine.options.DisablePrintRoute {
		debugPrintRoute(method, path, handlers)
	}

	//	TODO 待完善
}

func (engine *Engine) allocateContext() *app.RequestContext {
	ctx := engine.NewContext()
	ctx.Request.SetMaxKeepBodySize(engine.options.MaxKeepBodySize)
	ctx.Response.SetMaxKeepBodySize(engine.options.MaxKeepBodySize)
	ctx.SetClientIPFunc(engine.clientIPFunc)
	ctx.SetFormValueFunc(engine.formValueFunc)
	return ctx
}

func debugPrintRoute(httpMethod, absolutePath string, handlers app.HandlersChain) {
	nHandlers := len(handlers)
	handlerName := app.GetHandlerName(handlers.Last())
	if handlerName == "" {
		handlerName = utils.NameOfFunction(handlers.Last())
	}
	hlog.SystemLogger().Debugf("Method=%-6s absolutePath=%-25s --> handlerName=%s (num=%d handlers)", httpMethod, absolutePath, handlerName, nHandlers)
}

func initTrace(engine *Engine) stats.Level {
	for _, t := range engine.options.Tracers {
		if col, ok := t.(tracer.Tracer); ok {
			engine.tracerCtl.Append(col)
		}
	}

	if !engine.tracerCtl.HasTracer() {
		engine.enableTrace = false
	}

	traceLevel := stats.LevelDetailed
	if tl, ok := engine.options.TraceLevel.(stats.Level); ok {
		traceLevel = tl
	}
	return traceLevel
}

func NewEngine(opt *config.Options) *Engine {
	engine := &Engine{}
	if opt.TransporterNewer != nil {
		engine.transport = opt.TransporterNewer(opt)
	}
	engine.RouterGroup.engine = engine

	traceLevel := initTrace(engine)

	// 准备 RequestContext 池
	engine.ctxPool.New = func() any {
		ctx := engine.allocateContext()
		if engine.enableTrace {
			ti := traceinfo.NewTraceInfo()
			ti.Stats().SetLevel(traceLevel)
			ctx.SetTraceInfo(ti)
		}
		return ctx
	}

	// 初始化协议族
	//engine.protocolSuite = suite.New()

	return engine
}
