package route

import (
	"context"
	"path"
	"regexp"
	"strings"

	"github.com/favbox/gosky/wind/pkg/app"
	"github.com/favbox/gosky/wind/pkg/protocol/consts"
	rConsts "github.com/favbox/gosky/wind/pkg/route/consts"
)

// Route 表示请求路由的信息，包括请求方法、路径及其处理程序。
type Route struct {
	Method      string
	Path        string
	Handler     string
	HandlerFunc app.HandlerFunc
}

// Routes 定义了一组路由信息。
type Routes []Route

// Router 定义了所有路由处理器的接口。
type Router interface {
	Use(...app.HandlerFunc) Router
	Handle(string, string, ...app.HandlerFunc) Router
	Any(string, ...app.HandlerFunc) Router
	GET(string, ...app.HandlerFunc) Router
	POST(string, ...app.HandlerFunc) Router
	DELETE(string, ...app.HandlerFunc) Router
	PATCH(string, ...app.HandlerFunc) Router
	PUT(string, ...app.HandlerFunc) Router
	OPTIONS(string, ...app.HandlerFunc) Router
	HEAD(string, ...app.HandlerFunc) Router
	StaticFile(string, string) Router
	Static(string, string) Router
	StaticFS(string, *app.FS) Router
}

// Routers 定义了所有路由处理器的接口，包括单个路由及分组配置。
type Routers interface {
	Router
	Group(string, ...app.HandlerFunc) *RouterGroup
}

// RouterGroup 表示一个路由组，由前缀路径和一组处理器（中间件）组成。
type RouterGroup struct {
	Handlers app.HandlersChain
	basePath string
	engine   *Engine
	root     bool
}

func init() {
	g := RouterGroup{}
	g.Use()
}

var _ Routers = (*RouterGroup)(nil)

// Use 添加中间件到路由组。
func (group *RouterGroup) Use(middleware ...app.HandlerFunc) Router {
	group.Handlers = append(group.Handlers, middleware...)
	return group.returnObj()
}

var upperLetterReg = regexp.MustCompile("^[A-Z]+$")

// Handle 注册一个指定方法和路径的请求处理函数或中间件。
// 最后一个处理函数应为真正函数，其余函数应可共享于不同中间件。
//
// 对于 GET, POST, DELETE, PATCH, PUT, OPTIONS 和 HEAD 请求，可使用对应的快捷函数。
//
// 该函数为请求处理的通用函数，也可用于低频或非标的自定义请求方法（如：与代理的内部通信）。
func (group *RouterGroup) Handle(httpMethod string, relativePath string, handlers ...app.HandlerFunc) Router {
	if matches := upperLetterReg.MatchString(httpMethod); !matches {
		panic("http 请求方法 `" + httpMethod + "` 无效")
	}
	return group.handle(httpMethod, relativePath, handlers)
}

// Any 注册与所有方法都匹配的路由。
// GET, POST, PUT, PATCH, HEAD, OPTIONS, DELETE, CONNECT, TRACE。
func (group *RouterGroup) Any(relativePath string, handlers ...app.HandlerFunc) Router {
	group.handle(consts.MethodGet, relativePath, handlers)
	group.handle(consts.MethodPost, relativePath, handlers)
	group.handle(consts.MethodPut, relativePath, handlers)
	group.handle(consts.MethodPatch, relativePath, handlers)
	group.handle(consts.MethodHead, relativePath, handlers)
	group.handle(consts.MethodOptions, relativePath, handlers)
	group.handle(consts.MethodDelete, relativePath, handlers)
	group.handle(consts.MethodConnect, relativePath, handlers)
	group.handle(consts.MethodTrace, relativePath, handlers)
	return group.returnObj()
}

// GET 是 Handle("GET", relativePath, handlers) 的快捷方式。
func (group *RouterGroup) GET(relativePath string, handlers ...app.HandlerFunc) Router {
	return group.handle(consts.MethodGet, relativePath, handlers)
}

// POST 是 Handle("POST", relativePath, handlers) 的快捷方式。
func (group *RouterGroup) POST(relativePath string, handlers ...app.HandlerFunc) Router {
	return group.handle(consts.MethodPost, relativePath, handlers)
}

// DELETE 是 Handle("DELETE", relativePath, handlers) 的快捷方式。
func (group *RouterGroup) DELETE(relativePath string, handlers ...app.HandlerFunc) Router {
	return group.handle(consts.MethodDelete, relativePath, handlers)

}

// PATCH 是 Handle("PATCH", relativePath, handlers) 的快捷方式。
func (group *RouterGroup) PATCH(relativePath string, handlers ...app.HandlerFunc) Router {
	return group.handle(consts.MethodHead, relativePath, handlers)
}

// PUT 是 Handle("PUT", relativePath, handlers) 的快捷方式。
func (group *RouterGroup) PUT(relativePath string, handlers ...app.HandlerFunc) Router {
	return group.handle(consts.MethodHead, relativePath, handlers)
}

// OPTIONS 是 Handle("OPTIONS", relativePath, handlers) 的快捷方式。
func (group *RouterGroup) OPTIONS(relativePath string, handlers ...app.HandlerFunc) Router {
	return group.handle(consts.MethodHead, relativePath, handlers)
}

// HEAD 是 Handle("HEAD", relativePath, handlers) 的快捷方式。
func (group *RouterGroup) HEAD(relativePath string, handlers ...app.HandlerFunc) Router {
	return group.handle(consts.MethodHead, relativePath, handlers)
}

// StaticFile 注册一个提供静态文件处理的路由。
// StaticFile("favicon.ico", "./resources/favicon.ico")
func (group *RouterGroup) StaticFile(relativePath string, filepath string) Router {
	if strings.Contains(relativePath, ":") || strings.Contains(relativePath, "*") {
		panic("提供静态文件服务时不能使用 URL 参数，如':*'")
	}
	handler := func(c context.Context, ctx *app.RequestContext) {
		ctx.File(filepath)
	}
	group.GET(relativePath, handler)
	group.HEAD(relativePath, handler)
	return group.returnObj()
}

func (group *RouterGroup) Static(relativePath string, root string) Router {
	//TODO implement me
	panic("implement me")
}

func (group *RouterGroup) StaticFS(relativePath string, fs *app.FS) Router {
	//TODO implement me
	panic("implement me")
}

func (group *RouterGroup) Group(relativePath string, handlers ...app.HandlerFunc) *RouterGroup {
	//TODO implement me
	panic("implement me")
}

func (group *RouterGroup) returnObj() Routers {
	if group.root {
		return group.engine
	}
	return group
}

func (group *RouterGroup) handle(httpMethod, relativePath string, handlers app.HandlersChain) Router {
	absolutePath := group.calculateAbsolutePath(relativePath)
	handlers = group.combineHandlers(handlers)
	group.engine.addRoute(httpMethod, absolutePath, handlers)
	return group.returnObj()
}

func (group *RouterGroup) calculateAbsolutePath(relativePath string) string {
	return joinPaths(group.basePath, relativePath)
}

// 合并给定的处理链至当前路由组。
// 注意：若合并后的长度超过 consts.AbortIndex 会引发恐慌。
func (group *RouterGroup) combineHandlers(handlers app.HandlersChain) app.HandlersChain {
	finalSize := len(group.Handlers) + len(handlers)
	if finalSize >= int(rConsts.AbortIndex) {
		panic("处理函数过多")
	}
	mergedHandlers := make(app.HandlersChain, finalSize)
	copy(mergedHandlers, group.Handlers)
	copy(mergedHandlers[len(group.Handlers):], handlers)
	return mergedHandlers
}

func joinPaths(absolutePath, relativePath string) string {
	if relativePath == "" {
		return absolutePath
	}

	finalPath := path.Join(absolutePath, relativePath)
	appendSlash := lastChar(relativePath) == '/' && lastChar(finalPath) != '/'
	if appendSlash {
		return finalPath + "/"
	}
	return finalPath
}

func lastChar(s string) uint8 {
	if s == "" {
		panic("字符串长度不能为 0")
	}
	return s[len(s)-1]
}
