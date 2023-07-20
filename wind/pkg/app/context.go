package app

import (
	"github.com/favbox/gosky/wind/internal/bytestr"
	"github.com/favbox/gosky/wind/pkg/common/errors"
	"github.com/favbox/gosky/wind/pkg/network"
	"github.com/favbox/gosky/wind/pkg/protocol"
)

type Handler interface {
	ServeHTTP()
}

type (
	ClientIP        func(ctx *RequestContext) string
	ClientIPOptions struct {
		RemoteIPHeaders []string
		TrustedProxies  map[string]bool
	}
)

var defaultClientIPOptions = ClientIPOptions{
	RemoteIPHeaders: []string{"X-Real-IP", "X-Forward-For"},
	TrustedProxies:  map[string]bool{"0.0.0.0": true},
}

// ClientIPWithOption 用于生成 ClientIP 函数，并有 engine.SetClientIPFunc 设置。
func ClientIPWithOption(opts ClientIPOptions) ClientIP {
	return func(ctx *RequestContext) string {
		return ""
	}
}

// RequestContext 表示一个请求的上下文信息。
type RequestContext struct {
	conn     network.Conn
	Request  protocol.Request
	Response protocol.Response

	// 是附加到所有使用该上下文的处理器/中间件的错误列表。
	Errors errors.ErrorChain
}

func (ctx *RequestContext) AbortWithMsg(msg string, statusCode int) {
	ctx.Response.Reset()
	ctx.SetStatusCode(statusCode)
	ctx.SetContentTypeBytes(bytestr.DefaultContentType)
	ctx.SetBodyString(msg)
	ctx.Abort()
}

// File 快速写入指定文件到响应的主体流。
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

// SetContentTypeBytes 设置响应的内容类型标头值。
func (ctx *RequestContext) SetContentTypeBytes(contentType []byte) {
	ctx.Response.Header.SetContentTypeBytes(contentType)
}
