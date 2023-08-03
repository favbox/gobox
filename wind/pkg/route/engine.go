package route

import (
	"context"
	"fmt"
	"html/template"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/favbox/gosky/wind/internal/bytesconv"
	"github.com/favbox/gosky/wind/internal/bytestr"
	"github.com/favbox/gosky/wind/internal/nocopy"
	"github.com/favbox/gosky/wind/pkg/app"
	"github.com/favbox/gosky/wind/pkg/app/server/render"
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
	"github.com/favbox/gosky/wind/pkg/protocol/consts"
	"github.com/favbox/gosky/wind/pkg/protocol/http1"
	"github.com/favbox/gosky/wind/pkg/protocol/http1/factory"
	"github.com/favbox/gosky/wind/pkg/protocol/suite"
)

const unknownTransporterName = "unknown"

const (
	_ uint32 = iota
	statusInitialized
	statusRunning
	statusShutdown
	statusClosed
)

var (
	// 默认网络传输器（基于标准库实现，另外可选 netpoll.NewTransporter）
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

type hijackConn struct {
	network.Conn
	e *Engine
}

type Engine struct {
	noCopy nocopy.NoCopy

	// 引擎名称
	Name       string
	serverName atomic.Value

	// 路由器和协议服务器的配置项
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

	delims     render.Delims     // HTML 模板的分隔符
	funcMap    template.FuncMap  // HTML 模板的函数映射
	htmlRender render.HTMLRender // HTML 模板的渲染器

	// 是否不用劫持连接池来获取和释放劫持连接？
	//
	// 如果难以保证劫持连接不会被重复关闭，请设置为 true。
	NoHijackConnPool bool
	hijackConnPool   sync.Pool
	// 是否在处理劫持连接后继续保留该链接？
	// 这可节省协程，例如：当 wind 升级 http 连接 为 websocket 且
	// 连接已经转至另一个处理器，该处理器可按需关闭它。
	KeepHijackedConns bool

	// 底层传输的网络库，现有 go net 和 netpoll l两个选择
	transport network.Transporter

	// 链路追踪
	tracerCtl   tracer.Controller
	enableTrace bool

	// 管理协议层不同协议对应的服务器的创建
	protocolSuite         *suite.Config
	protocolServers       map[string]protocol.Server       // 协议与普通服务器的映射
	protocolStreamServers map[string]protocol.StreamServer // 协议与流式服务器的映射

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

	// 用于表示引擎状态（Init/Running/Shutdown/Closed）。
	status uint32

	// OnRun 是引擎启动时，依次触发的一组钩子函数。
	OnRun []CtxErrCallback

	// OnShutdown 是引擎关闭时，同时触发的一组钩子函数。
	OnShutdown []CtxCallback

	// 自定义获取客户端 IP 的函数。
	clientIPFunc app.ClientIP
	// 自定义获取表单值的函数。
	formValueFunc app.FormValueFunc
}

// Init 初始化路由引擎。
//
// 如添加默认的 http1 协议服务器。
func (engine *Engine) Init() error {
	// 默认添加内置的 http1 服务器
	if !engine.HasServer(suite.HTTP1) {
		engine.AddProtocol(suite.HTTP1, factory.NewServerFactory(newHttp1OptionFromEngine(engine)))
	}

	serverMap, streamServerMap, err := engine.protocolSuite.LoadAll(engine)
	if err != nil {
		return errs.New(err, errs.ErrorTypePrivate, "加载所有协议组错误")
	}

	engine.protocolServers = serverMap
	engine.protocolStreamServers = streamServerMap

	// 若启用 ALPN 协议自动切换，则将 suite.HTTP1 作为写一个备用协议。
	if engine.alpnEnable() {
		engine.options.TLS.NextProtos = append(engine.options.TLS.NextProtos, suite.HTTP1)
	}

	if !atomic.CompareAndSwapUint32(&engine.status, 0, statusInitialized) {
		return errInitFailed
	}
	return nil
}

// NewContext 创建一个无请求/响应的纯粹请求上下文。
func (engine *Engine) NewContext() *app.RequestContext {
	return app.NewContext(engine.maxParams)
}

func (engine *Engine) IsRunning() bool {
	return atomic.LoadUint32(&engine.status) == statusRunning
}

// IsStreamRequestBody 是否流式处理请求正文？
func (engine *Engine) IsStreamRequestBody() bool {
	return engine.options.StreamRequestBody
}

// IsTraceEnable 是否启用链路跟踪？
func (engine *Engine) IsTraceEnable() bool {
	return engine.enableTrace
}

// NoRoute 设置 404 请求方法未找到时对应的处理链，默认返回 404 状态码。
func (engine *Engine) NoRoute(handlers ...app.HandlerFunc) {
	engine.noRoute = handlers
	engine.rebuild404Handlers()
}

func (engine *Engine) rebuild404Handlers() {
	engine.allNoRoute = engine.combineHandlers(engine.noRoute)
}

// NoMethod 设置405请求方法不允许时对应的处理链。
func (engine *Engine) NoMethod(handlers ...app.HandlerFunc) {
	engine.noMethod = handlers
	engine.rebuild405Handlers()
}

func (engine *Engine) rebuild405Handlers() {
	engine.allNoMethod = engine.combineHandlers(engine.noMethod)
}

// PrintRoute 递归打印给定方法的路由节点信息。
func (engine *Engine) PrintRoute(method string) {
	root := engine.trees.get(method)
	printNode(root.root, 0)
}

// 递归打印路由节点信息
func printNode(node *node, level int) {
	fmt.Println("node.prefix: " + node.prefix)
	fmt.Println("node.ppath: " + node.ppath)
	fmt.Printf("level: %#v\n\n", level)
	for i := 0; i < len(node.children); i++ {
		printNode(node.children[i], level+1)
	}
}

// Routes 返回已注册的路由切片，及关键信息，如： HTTP 方法、路径和处理器名称。
func (engine *Engine) Routes() (routes Routes) {
	for _, tree := range engine.trees {
		routes = iterate(tree.method, routes, tree.root)
	}
	return routes
}

func iterate(method string, routes Routes, root *node) Routes {
	if len(root.handlers) > 0 {
		handlerFunc := root.handlers.Last()
		routes = append(routes, Route{
			Method:      method,
			Path:        root.ppath,
			Handler:     utils.NameOfFunction(handlerFunc),
			HandlerFunc: handlerFunc,
		})
	}

	for _, child := range root.children {
		routes = iterate(method, routes, child)
	}

	if root.paramChild != nil {
		routes = iterate(method, routes, root.paramChild)
	}

	if root.anyChild != nil {
		routes = iterate(method, routes, root.anyChild)
	}

	return routes
}

// LoadHTMLFiles 加载一组 HTML 文件，并关联到 HTML 渲染器。
func (engine *Engine) LoadHTMLFiles(files ...string) {
	tmpl := template.Must(template.New("").
		Delims(engine.delims.Left, engine.delims.Right).
		Funcs(engine.funcMap).
		ParseFiles(files...))

	if engine.options.AutoReloadRender {
		engine.SetAutoReloadHTMLTemplate(tmpl, files)
		return
	}

	engine.SetHTMLTemplate(tmpl)
}

// LoadHTMLGlob 加载给定 pattern 模式的 HTML 文件，并关联到 HTML 渲染器。
func (engine *Engine) LoadHTMLGlob(pattern string) {
	tmpl := template.Must(template.New("").
		Delims(engine.delims.Left, engine.delims.Right).
		Funcs(engine.funcMap).
		ParseGlob(pattern))

	if engine.options.AutoReloadRender {
		files, err := filepath.Glob(pattern)
		if err != nil {
			hlog.SystemLogger().Errorf("LoadHTMLGlob: %v", err)
			return
		}
		engine.SetAutoReloadHTMLTemplate(tmpl, files)
		return
	}

	engine.SetHTMLTemplate(tmpl)
}

// SetAutoReloadHTMLTemplate 关联模板与调试环境的 HTML 模板渲染器。
func (engine *Engine) SetAutoReloadHTMLTemplate(tmpl *template.Template, files []string) {
	engine.htmlRender = &render.HTMLDebug{
		Template:        tmpl,
		Files:           files,
		FuncMap:         engine.funcMap,
		Delims:          engine.delims,
		RefreshInterval: engine.options.AutoReloadInterval,
	}
}

// SetHTMLTemplate 关联模板与生产环境的 HTML 渲染器。
func (engine *Engine) SetHTMLTemplate(tmpl *template.Template) {
	engine.htmlRender = render.HTMLProduction{
		Template: tmpl.Funcs(engine.funcMap),
	}
}

// 让路由引擎实现处理器接口。
func (engine *Engine) ServeHTTP(c context.Context, ctx *app.RequestContext) {
	if engine.PanicHandler != nil {
		defer engine.recover(ctx)
	}

	rPath := string(ctx.Request.URI().Path())
	httpMethod := bytesconv.B2s(ctx.Request.Header.Method())
	unescape := false
	if engine.options.UseRawPath {
		rPath = string(ctx.Request.URI().PathOriginal())
		unescape = engine.options.UnescapePathValues
	}

	if engine.options.RemoveExtraSlash {
		rPath = utils.CleanPath(rPath)
	}

	// 遵循 RFC7230#section-5.3
	if rPath == "" || rPath[0] != '/' {
		serveError(c, ctx, consts.StatusBadRequest, default400Body)
		return
	}

	// 查找给定 HTTP 方法树的根
	t := engine.trees
	paramsPointer := &ctx.Params
	for i, tl := 0, len(t); i < tl; i++ {
		if t[i].method != httpMethod {
			continue
		}
		// 在树中查找路由
		value := t[i].find(rPath, paramsPointer, unescape)

		if value.handlers != nil {
			ctx.SetHandlers(value.handlers)
			ctx.SetFullPath(value.fullPath)
			ctx.Next(c)
			return
		}
		if httpMethod != consts.MethodConnect && rPath != "/" {
			if value.tsr && engine.options.RedirectTrailingSlash {
				redirectTrailingSlash(ctx)
				return
			}
			if engine.options.RedirectFixedPath && redirectFixedPath(ctx, t[i].root, engine.options.RedirectFixedPath) {
				return
			}
		}
		break
	}

	if engine.options.HandleMethodNotAllowed {
		for _, tree := range engine.trees {
			if tree.method == httpMethod {
				continue
			}
			if value := tree.find(rPath, paramsPointer, unescape); value.handlers != nil {
				ctx.SetHandlers(engine.allNoMethod)
				serveError(c, ctx, consts.StatusMethodNotAllowed, default405Body)
				return
			}
		}
	}
	ctx.SetHandlers(engine.allNoRoute)
	serveError(c, ctx, consts.StatusNotFound, default400Body)
}

// AddProtocol 添加给定协议的服务器工厂方法。
func (engine *Engine) AddProtocol(protocol string, factory any) {
	engine.protocolSuite.Add(protocol, factory)
}

// Close 关闭路由引擎。
//
// 包括底层传输器、HTML 渲染器可能用到的文件监视器。
func (engine *Engine) Close() error {
	if engine.htmlRender != nil {
		engine.htmlRender.Close()
	}
	return engine.transport.Close()
}

// Delims 设置 HTML 模板的左右分隔符并返回引擎。
func (engine *Engine) Delims(left, right string) *Engine {
	engine.delims = render.Delims{
		Left:  left,
		Right: right,
	}
	return engine
}

// GetCtxPool 返回引擎的上下文池子。
func (engine *Engine) GetCtxPool() *sync.Pool {
	return &engine.ctxPool
}

// GetOptions 返回路由器和协议服务器的配置项。
func (engine *Engine) GetOptions() *config.Options {
	return engine.options
}

func (engine *Engine) GetServerName() []byte {
	v := engine.serverName.Load()
	var serverName []byte
	if v == nil {
		serverName = []byte(engine.Name)
		if len(serverName) == 0 {
			serverName = bytestr.DefaultServerName
		}
		engine.serverName.Store(serverName)
	} else {
		serverName = v.([]byte)
	}
	return serverName
}

// GetTracer 获取链路跟踪控制器。
func (engine *Engine) GetTracer() tracer.Controller {
	return engine.tracerCtl
}

// GetTransporterName 获取底层网络传输器的名称。
func (engine *Engine) GetTransporterName() string {
	return getTransporterName(engine.transport)
}

// HasServer 判断给定协议的普通服务器工厂是否存在？
func (engine *Engine) HasServer(name string) bool {
	return engine.protocolSuite.Get(name) != nil
}

// HijackConnHandle 处理给定的劫持连接。
func (engine *Engine) HijackConnHandle(c network.Conn, h app.HijackHandler) {
	engine.hijackConnHandle(c, h)
}

// SetClientIPFunc 设置获取客户端 IP 的自定义函数。
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

func (engine *Engine) alpnEnable() bool {
	return engine.options.TLS != nil && engine.options.ALPN
}

// 处理恐慌。
func (engine *Engine) recover(ctx *app.RequestContext) {
	if r := recover(); r != nil {
		engine.PanicHandler(context.Background(), ctx)
	}
}

// 分配一个限定
func (engine *Engine) allocateContext() *app.RequestContext {
	ctx := engine.NewContext()
	ctx.Request.SetMaxKeepBodySize(engine.options.MaxKeepBodySize)
	ctx.Response.SetMaxKeepBodySize(engine.options.MaxKeepBodySize)
	ctx.SetClientIPFunc(engine.clientIPFunc)
	ctx.SetFormValueFunc(engine.formValueFunc)
	return ctx
}

// 处理劫持连接。
func (engine *Engine) hijackConnHandle(c network.Conn, h app.HijackHandler) {
	hjc := engine.acquireHijackConn(c)
	h(hjc)

	if !engine.KeepHijackedConns {
		c.Close()
		engine.releaseHijackConn(hjc)
	}
}

// 获取劫持连接。
func (engine *Engine) acquireHijackConn(c network.Conn) *hijackConn {
	// 不用劫持连接池
	if engine.NoHijackConnPool {
		return &hijackConn{
			Conn: c,
			e:    engine,
		}
	}

	// 用连接池
	v := engine.hijackConnPool.Get()

	// 但是还没有可用实例，返回一个新实例
	if v == nil {
		return &hijackConn{
			Conn: c,
			e:    engine,
		}
	}

	// 池中有可用实例，则更新连接
	hjc := v.(*hijackConn)
	hjc.Conn = c
	return hjc
}

// 释放劫持连接。
func (engine *Engine) releaseHijackConn(hjc *hijackConn) {
	if engine.NoHijackConnPool {
		return
	}
	hjc.Conn = nil
	engine.hijackConnPool.Put(hjc)
}

func debugPrintRoute(httpMethod, absolutePath string, handlers app.HandlersChain) {
	nHandlers := len(handlers)
	handlerName := app.GetHandlerName(handlers.Last())
	if handlerName == "" {
		handlerName = utils.NameOfFunction(handlers.Last())
	}
	hlog.SystemLogger().Debugf("Method=%-6s absolutePath=%-25s --> handlerName=%s (num=%d handlers)", httpMethod, absolutePath, handlerName, nHandlers)
}

func getTransporterName(transporter network.Transporter) string {
	defer func() {}()
	t := reflect.ValueOf(transporter).Type().String()
	return strings.Split(strings.TrimPrefix(t, "*"), ".")[0]
}

// 仅用于内置的 http1 实现。
func newHttp1OptionFromEngine(engine *Engine) *http1.Option {
	opt := &http1.Option{
		StreamRequestBody:            engine.options.StreamRequestBody,
		GetOnly:                      engine.options.GetOnly,
		DisablePreParseMultipartForm: engine.options.DisablePreParseMultipartForm,
		DisableKeepalive:             engine.options.DisableKeepalive,
		NoDefaultServerHeader:        engine.options.NoDefaultServerHeader,
		MaxRequestBodySize:           engine.options.MaxRequestBodySize,
		IdleTimeout:                  engine.options.IdleTimeout,
		ReadTimeout:                  engine.options.ReadTimeout,
		ServerName:                   engine.GetServerName(),
		TLS:                          engine.options.TLS,
		EnableTrace:                  engine.IsTraceEnable(),
		HTMLRender:                   engine.htmlRender,
		ContinueHandler:              engine.ContinueHandler,
		HijackConnHandle:             engine.HijackConnHandle,
	}
	// 标准库的空闲超时必不能为零，若为 0 则置为 -1。
	// 由于网络库的触发方式不同，具体原因请参阅该值的实际使用情况。
	if opt.IdleTimeout == 0 && engine.GetTransporterName() == "standard" {
		opt.IdleTimeout = -1
	}
	return opt
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

func redirectFixedPath(ctx *app.RequestContext, root *node, fixTrailingSlash bool) bool {
	rPath := bytesconv.B2s(ctx.Request.URI().Path())
	if fixedPath, ok := root.findCaseInsensitivePath(utils.CleanPath(rPath), fixTrailingSlash); ok {
		ctx.Request.SetRequestURI(bytesconv.B2s(fixedPath))
		redirectRequest(ctx)
		return true
	}
	return false
}

func redirectTrailingSlash(ctx *app.RequestContext) {
	p := bytesconv.B2s(ctx.Request.URI().Path())
	if prefix := utils.CleanPath(bytesconv.B2s(ctx.Request.Header.Peek("X-Forwarded-Prefix"))); prefix != "." {
		p = prefix + "/" + p
	}

	tmpURI := trailingSlashURL(p)

	query := ctx.Request.URI().QueryString()

	if len(query) > 0 {
		tmpURI = tmpURI + "?" + bytesconv.B2s(query)
	}

	ctx.Request.SetRequestURI(tmpURI)
	redirectRequest(ctx)
}

func redirectRequest(ctx *app.RequestContext) {
	code := consts.StatusMovedPermanently // 永久跳转，GET 请求
	if bytesconv.B2s(ctx.Request.Header.Method()) != consts.MethodGet {
		code = consts.StatusTemporaryRedirect
	}

	ctx.Redirect(code, ctx.Request.URI().RequestURI())
}

func trailingSlashURL(ts string) string {
	tmpURI := ts + "/"
	if length := len(ts); length > 1 && ts[length-1] == '/' {
		tmpURI = ts[:length-1]
	}
	return tmpURI
}

func serveError(c context.Context, ctx *app.RequestContext, code int, defaultMessage []byte) {
	ctx.SetStatusCode(code)
	ctx.Next(c)
	if ctx.Response.StatusCode() == code {
		// 若正文存在（或由用户定制），别管他。
		if ctx.Response.HasBodyBytes() || ctx.Response.IsBodyStream() {
			return
		}
		ctx.Response.Header.Set(consts.HeaderContentType, consts.MIMETextPlain)
		ctx.Response.SetBody(defaultMessage)
	}
}

func NewEngine(opt *config.Options) *Engine {
	engine := &Engine{
		trees: make(MethodTrees, 0, 9),
		RouterGroup: RouterGroup{
			Handlers: nil,
			basePath: opt.BasePath,
			root:     true,
		},
		transport: defaultTransporter(opt),
	}
	if opt.TransporterNewer != nil {
		engine.transport = opt.TransporterNewer(opt)
	}
	engine.RouterGroup.engine = engine

	traceLevel := initTrace(engine)

	// 定义 RequestContext 上下文池的新建函数
	engine.ctxPool.New = func() any {
		ctx := engine.allocateContext()
		if engine.enableTrace {
			ti := traceinfo.NewTraceInfo()
			ti.Stats().SetLevel(traceLevel)
			ctx.SetTraceInfo(ti)
		}
		return ctx
	}

	// 初始化协议组
	engine.protocolSuite = suite.New()

	return engine
}
