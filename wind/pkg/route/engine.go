package route

import (
	"sync/atomic"

	"github.com/favbox/gosky/wind/internal/nocopy"
	"github.com/favbox/gosky/wind/pkg/app"
	"github.com/favbox/gosky/wind/pkg/common/config"
	"github.com/favbox/gosky/wind/pkg/common/hlog"
	"github.com/favbox/gosky/wind/pkg/common/utils"
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

func debugPrintRoute(httpMethod, absolutePath string, handlers app.HandlerChain) {
	nHandlers := len(handlers)
	handlerName := app.GetHandlerName(handlers.Last())
	if handlerName == "" {
		handlerName = utils.NameOfFunction(handlers.Last())
	}
	hlog.SystemLogger().Debugf("Method=%-6s absolutePath=%-25s --> handlerName=%s (num=%d handlers)", httpMethod, absolutePath, handlerName, nHandlers)
}
