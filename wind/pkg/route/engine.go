package route

import (
	"sync"
	"sync/atomic"

	"github.com/favbox/gosky/wind/internal/nocopy"
	"github.com/favbox/gosky/wind/pkg/app"
	"github.com/favbox/gosky/wind/pkg/common/config"
	"github.com/favbox/gosky/wind/pkg/common/hlog"
	"github.com/favbox/gosky/wind/pkg/common/tracer"
	"github.com/favbox/gosky/wind/pkg/common/tracer/stats"
	"github.com/favbox/gosky/wind/pkg/common/tracer/traceinfo"
	"github.com/favbox/gosky/wind/pkg/common/utils"
	"github.com/favbox/gosky/wind/pkg/network"
)

const unknownTransporterName = "unknown"

var (
	defaultTransporter = standard.NewTransporter
)

type Engine struct {
	noCopy nocopy.NoCopy

	// 引擎名称
	Name       string
	serverName atomic.Value

	// 路由和协议服务器的配置项
	options *config.Options

	// 路由
	RouterGroup

	// 最大路由参数个数
	maxParams uint16

	// 底层网络传输器
	transport network.Transporter

	// 跟踪控制器
	tracerCtl   tracer.Controller
	enableTrace bool

	// 请求上下文池
	ctxPool sync.Pool

	// 自定义函数
	clientIPFunc  app.ClientIP
	formValueFunc app.FormValueFunc
}

// NewContext 创建一个无请求/响应的纯粹请求上下文。
func (engine *Engine) NewContext() *app.RequestContext {
	return app.NewContext(engine.maxParams)
}

func (engine *Engine) SetClientIPFunc(f app.ClientIP) {
	engine.clientIPFunc = f
}

func (engine *Engine) addRoute(method, path string, handlers app.HandlerChain) {
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

func debugPrintRoute(httpMethod, absolutePath string, handlers app.HandlerChain) {
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
	engine.protocolSuite = suite.New()

	return engine
}
