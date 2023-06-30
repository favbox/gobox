package config

import (
	"context"
	"crypto/tls"
	"net"
	"time"

	"github.com/favbox/gobox/hertz/pkg/network"
)

const (
	defaultKeepAliveTimeout   = 1 * time.Minute
	defaultReadTimeout        = 3 * time.Minute
	defaultAddr               = ":8888"
	defaultNetwork            = "tcp"
	defaultBasePath           = "/"
	defaultMaxRequestBodySize = 4 * 1024 * 1024
	defaultWaitExitTimeout    = 5 * time.Second
	defaultReadBufferSize     = 4 * 1024
)

// Option 是唯一可用于设置 Options 的结构体。
type Option struct {
	F func(o *Options)
}

type Options struct {
	// 保活时长。当空闲连接超过这个时长，服务器会发送保活数据包以确保其为有效连接。
	//
	// 注意：通常无需关心该值，关心 IdleTimeout 即可。
	KeepAliveTimeout time.Duration

	// 从底层库读取的超时时间。
	ReadTimeout  time.Duration
	WriteTimeout time.Duration

	// 当 IdleTimeout 期间无请求时，服务器将关闭连接。
	// 默认为 ReadTimeout。零值意为不会超时。
	IdleTimeout time.Duration

	// 将 /foo/ 重定向到 /foo
	RedirectTrailingSlash bool

	// 将 /FOO 和 /..//FOO 重定向到 /foo
	RedirectFixedPath bool

	HandleMethodNotAllowed bool

	// 用 url.RawPath 来查找参数。
	UseRawPath bool

	// 移除额外的斜杠，以从URL中解析参数。
	RemoveExtraSlash bool

	// 不逃逸路径值。
	UnescapePathValues bool

	MaxRequestBodySize           int
	MaxKeepBodySize              int
	GetOnly                      bool
	DisableKeepalive             bool
	DisablePreParseMultipartForm bool
	StreamRequestBody            bool
	NoDefaultServerHeader        bool
	DisablePrintRoute            bool
	Network                      string
	Addr                         string
	BasePath                     string
	ExitWaitTimeout              time.Duration
	TLS                          *tls.Config
	ALPN                         bool
	H2C                          bool
	ReadBufferSize               int
	Tracers                      []any
	TraceLevel                   any
	ListenConfig                 *net.ListenConfig

	// 是创建传输器的函数。
	TransporterNewer    func(opt *Options) network.Transporter
	AltTransporterNewer func(opt *Options) network.Transporter

	// 在 netpoll 中，OnAccept 是在接受连接之后且加到 epoll 之前调用的。OnConnect 是在加到 epoll 之后调用的。
	// 区别在于 OnConnect 能取数据，而 OnAccept 不能。例如想检查对端IP是否在黑名单中，可使用 OnAccept。
	//
	// 在 go/net 中，OnAccept 是在接受连接之后且建立 tls 连接之前调用的。建立 tls 连接后执行 OnConnect。
	OnAccept  func(conn net.Conn) context.Context
	OnConnect func(ctx context.Context, conn network.Conn) context.Context

	// TODO 服务注册相关

	// 启用 HTML 模板自动重载机制
	AutoReloadRender bool
	// HTML 模板自动重载时间间隔。
	// 默认为0，即根据文件变更事件立即重载。
	AutoReloadInterval time.Duration
}

func (o *Options) Apply(opts []Option) {
	for _, opt := range opts {
		opt.F(o)
	}
}

func NewOptions(opts []Option) *Options {
	options := &Options{
		KeepAliveTimeout:      defaultKeepAliveTimeout,
		ReadTimeout:           defaultReadTimeout,
		IdleTimeout:           defaultReadTimeout,
		RedirectTrailingSlash: true,
		UnescapePathValues:    true,
		DisablePrintRoute:     false,
		Network:               defaultNetwork,
		Addr:                  defaultAddr,
		BasePath:              defaultBasePath,
		MaxRequestBodySize:    defaultMaxRequestBodySize,
		MaxKeepBodySize:       defaultMaxRequestBodySize,
		ExitWaitTimeout:       defaultWaitExitTimeout,
		ReadBufferSize:        defaultReadBufferSize,
		Tracers:               []any{},
		TraceLevel:            new(any),
	}
	options.Apply(opts)
	return options
}
