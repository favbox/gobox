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

type RequestContext struct {
}
