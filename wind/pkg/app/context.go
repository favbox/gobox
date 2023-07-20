package app

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
}

// File 快速将指定文件写入响应的主体流。
func (ctx *RequestContext) File(filepath string) {
	ServeFile(ctx, filepath)
}
