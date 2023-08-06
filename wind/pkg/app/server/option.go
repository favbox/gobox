package server

import (
	"strings"

	"github.com/favbox/gosky/wind/pkg/common/config"
)

// WithHostPorts 设置监听地址。
func WithHostPorts(addr string) config.Option {
	return config.Option{F: func(o *config.Options) {
		o.Addr = addr
	}}
}

// WithMaxRequestBodySize 限制请求的正文最大字节数。
//
// 大于此大小的正文缓冲区将被放回缓冲池。
func WithMaxRequestBodySize(bs int) config.Option {
	return config.Option{F: func(o *config.Options) {
		o.MaxRequestBodySize = bs
	}}
}

// WithUseRawPath 启用 url.RawPath 来查找参数。
func WithUseRawPath(b bool) config.Option {
	return config.Option{F: func(o *config.Options) {
		o.UseRawPath = b
	}}
}

// WithBasePath 设置基础路径。自动补全前缀和后缀的 "/"。
func WithBasePath(basePath string) config.Option {
	return config.Option{F: func(o *config.Options) {
		// 必须以 "/" 作为前缀和后缀，否则就拼接上 "/"
		if !strings.HasPrefix(basePath, "/") {
			basePath = "/" + basePath
		}
		if !strings.HasSuffix(basePath, "/") {
			basePath = basePath + "/"
		}
		o.BasePath = basePath
	}}
}

// WithStreamBody 确定是否在流中读取正文。
//
// 启用流式处理，可在请求的正文超过当前字节数限制时，更快地调用处理器。
func WithStreamBody(b bool) config.Option {
	return config.Option{F: func(o *config.Options) {
		o.StreamRequestBody = b
	}}
}
