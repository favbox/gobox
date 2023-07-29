package app

import (
	"io"
	"mime/multipart"
	"time"

	"github.com/favbox/gosky/wind/internal/bytesconv"
	"github.com/favbox/gosky/wind/internal/bytestr"
	"github.com/favbox/gosky/wind/pkg/common/errors"
	"github.com/favbox/gosky/wind/pkg/common/tracer/traceinfo"
	"github.com/favbox/gosky/wind/pkg/network"
	"github.com/favbox/gosky/wind/pkg/protocol"
	"github.com/favbox/gosky/wind/pkg/protocol/consts"
	rConsts "github.com/favbox/gosky/wind/pkg/route/consts"
	"github.com/favbox/gosky/wind/pkg/route/param"
)

type Handler interface {
	ServeHTTP()
}

// RequestContext 表示一个请求的上下文信息。
type RequestContext struct {
	conn     network.Conn
	Request  protocol.Request
	Response protocol.Response

	// 是附加到所有使用该上下文的处理器/中间件的错误列表。
	Errors errors.ErrorChain

	Params   param.Params
	handlers HandlersChain
	fullPath string
	index    int8 // 该请求处理链的当前索引

	// 跟踪信息
	traceInfo traceinfo.TraceInfo

	// 是否启用跟踪
	enableTrace bool

	// 通过自定义函数获取客户端 IP
	clientIPFunc ClientIP

	// 通过自定义函数获取表单值
	formValueFunc FormValueFunc
}

// Abort 中止处理，并防止调用挂起的处理器。
//
// 注意该函数不会停止当前处理器。
// 假设身份鉴权中间件鉴权失败（如密码不匹配），调用 Abort 可确保该请求的后续处理器不被调用。
func (ctx *RequestContext) Abort() {
	ctx.index = rConsts.AbortIndex
}

// AbortWithStatus 设置状态码并中止处理。
//
// 例如，对于身份鉴权失败的请求可使用：ctx.AbortWithStatus(401)
func (ctx *RequestContext) AbortWithStatus(code int) {
	ctx.SetStatusCode(code)
	ctx.Abort()
}

// AbortWithMsg 设置响应体和状态码，并中止响应。
//
// 警告：将重置先前的响应。
func (ctx *RequestContext) AbortWithMsg(msg string, statusCode int) {
	ctx.Response.Reset()
	ctx.SetStatusCode(statusCode)
	ctx.SetContentTypeBytes(bytestr.DefaultContentType)
	ctx.SetBodyString(msg)
	ctx.Abort()
}

// File 快速写入指定文件到响应的正文流。
func (ctx *RequestContext) File(filepath string) {
	ServeFile(ctx, filepath)
}

// URI 返回请求的完整网址。
func (ctx *RequestContext) URI() *protocol.URI {
	return ctx.Request.URI()
}

// Path 返回请求的路径。
func (ctx *RequestContext) Path() []byte {
	return ctx.URI().Path()
}

// SetStatusCode 设置响应的状态码。
func (ctx *RequestContext) SetStatusCode(statusCode int) {
	ctx.Response.SetStatusCode(statusCode)
}

// SetContentType 设置响应的内容类型标头值。
func (ctx *RequestContext) SetContentType(contentType string) {
	ctx.Response.Header.SetContentType(contentType)
}

// SetContentTypeBytes 设置响应的内容类型标头值。
func (ctx *RequestContext) SetContentTypeBytes(contentType []byte) {
	ctx.Response.Header.SetContentTypeBytes(contentType)
}

// SetBodyStream 设置响应的正文流和大小（可选）。
func (ctx *RequestContext) SetBodyStream(bodyStream io.Reader, bodySize int) {
	ctx.Response.SetBodyStream(bodyStream, bodySize)
}

// SetBodyString 设置响应的主体。
func (ctx *RequestContext) SetBodyString(body string) {
	ctx.Response.SetBodyString(body)
}

// IfModifiedSince 如果 lastModified 超过请求标头中的 'If-Modified-Since' 值，则返回真。
//
// 若无此标头或日期解析失败也返回真。
func (ctx *RequestContext) IfModifiedSince(lastModified time.Time) bool {
	ifModStr := ctx.Request.Header.PeekIfModifiedSinceBytes()
	if len(ifModStr) == 0 {
		return true
	}
	ifMod, err := bytesconv.ParseHTTPDate(ifModStr)
	if err != nil {
		return true
	}
	lastModified = lastModified.Truncate(time.Second)
	return ifMod.Before(lastModified)
}

// NotModified 重置响应并将响应的状态码设置为 '304 Not Modified'。
func (ctx *RequestContext) NotModified() {
	ctx.Response.Reset()
	ctx.SetStatusCode(consts.StatusNotModified)
}

// IsHead 是否为 HEAD 请求？
func (ctx *RequestContext) IsHead() bool {
	return ctx.Request.Header.IsHead()
}

// Host 获取请求的主机地址。
func (ctx *RequestContext) Host() []byte {
	return ctx.URI().Host()
}

// WriteString 附加 s 到响应的主体。
func (ctx *RequestContext) WriteString(s string) (int, error) {
	ctx.Response.AppendBodyString(s)
	return len(s), nil
}

func (ctx *RequestContext) GetTraceInfo() traceinfo.TraceInfo {
	return ctx.traceInfo
}

func (ctx *RequestContext) SetTraceInfo(t traceinfo.TraceInfo) {
	ctx.traceInfo = t
}

func (ctx *RequestContext) IsEnableTrace() bool {
	return ctx.enableTrace
}

// SetEnableTrace 设置是否启用跟踪。
//
// 注意：业务处理程序不可修改此值，否则可能引起恐慌。
func (ctx *RequestContext) SetEnableTrace(enable bool) {
	ctx.enableTrace = enable
}

func (ctx *RequestContext) SetClientIPFunc(fn ClientIP) {
	ctx.clientIPFunc = fn
}

func (ctx *RequestContext) SetFormValueFunc(f FormValueFunc) {
	ctx.formValueFunc = f
}

// QueryArgs 返回请求 URL 中的查询参数。
//
// 不会返回 POST 请求的参数 - 请使用 PostArgs()。
// 其他参数请看 PostArgs, FormValue and FormFile。
func (ctx *RequestContext) QueryArgs() *protocol.Args {
	return ctx.URI().QueryArgs()
}

// PostArgs 返回 POST 参数。
func (ctx *RequestContext) PostArgs() *protocol.Args {
	return ctx.Request.PostArgs()
}

// FormFile 返回表单中指定 name 的第一个文件头。
func (ctx *RequestContext) FormFile(name string) (*multipart.FileHeader, error) {
	return ctx.Request.FormFile(name)
}

// FormValue 获取指定 key 的表单值。
//
// 查找位置：
//
//   - Query 查询字符串
//   - POST 或 PUT 正文
//
// 还有更多细粒度的方法可获取表单值：
//
//   - QueryArgs 用于获取查询字符串中的值。
//   - PostArgs 用于获取 POST 或 PUT 正文的值。
//   - MultipartForm 用于获取多部分表单的值。
//   - FormFile 用于获取上传的文件。
//
// 通过 engine.SetCustomFormValueFunc 可改变 FormValue 的取值行为。
func (ctx *RequestContext) FormValue(key string) []byte {
	if ctx.formValueFunc != nil {
		return ctx.formValueFunc(ctx, key)
	}
	return defaultFormValue(ctx, key)
}

// MultipartForm 返回请求的多部分表单。
//
// 若请求的内容类型不是 'multipart/form-data' 则返回 errors.ErrNoMultipartForm。
//
// 所有上传的临时文件都将被自动删除。如果你想保留上传的文件，可移动或复制到新位置。
//
// 使用 SaveMultipartFile 可持久化保存上传的文件。
//
// 另见 FormFile and FormValue.
func (ctx *RequestContext) MultipartForm() (*multipart.Form, error) {
	return ctx.Request.MultipartForm()
}

type (
	// ClientIP 自定义获取客户端 IP 的函数
	ClientIP        func(ctx *RequestContext) string
	ClientIPOptions struct {
		RemoteIPHeaders []string
		TrustedProxies  map[string]bool
	}

	// FormValueFunc 自定义获取表单值的函数
	FormValueFunc func(*RequestContext, string) []byte
)

var defaultFormValue = func(ctx *RequestContext, key string) []byte {
	v := ctx.QueryArgs().Peek(key)
	if len(v) > 0 {
		return v
	}
	v = ctx.PostArgs().Peek(key)
	if len(v) > 0 {
		return v
	}
	mf, err := ctx.MultipartForm()
	if err == nil && mf.Value != nil {
		vv := mf.Value[key]
		if len(vv) > 0 {
			return []byte(vv[0])
		}
	}
	return nil
}

var defaultClientIPOptions = ClientIPOptions{
	RemoteIPHeaders: []string{"X-Real-IP", "X-Forward-For"},
	TrustedProxies:  map[string]bool{"0.0.0.0": true},
}
var defaultClientIP = ClientIPWithOption(defaultClientIPOptions)

// ClientIPWithOption 用于生成 ClientIP 函数，并有 engine.SetClientIPFunc 设置。
func ClientIPWithOption(opts ClientIPOptions) ClientIP {
	return func(ctx *RequestContext) string {
		return ""
	}
}

// NewContext 创建一个指定最大路由参数的无请求/响应信息的纯粹 RequestContext。
func NewContext(maxParams uint16) *RequestContext {
	v := make(param.Params, 0, maxParams)
	ctx := &RequestContext{Params: v, index: -1}
	return ctx
}
