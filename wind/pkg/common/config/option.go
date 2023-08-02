package config

import (
	"context"
	"crypto/tls"
	"net"
	"time"

	"github.com/favbox/gosky/wind/pkg/app/server/registry"
	"github.com/favbox/gosky/wind/pkg/network"
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

// Option 是配置项 Options 唯一的配置方法结构体。
type Option struct {
	F func(o *Options)
}

// Options 是配置项的结构体。
type Options struct {
	// 长连接超时时长，默认 1 分钟。
	// 注意，通常无需关心该值，关心 IdleTimeout 即可。
	KeepAliveTimeout time.Duration

	// 底层库读取的超时时间，默认 3 分钟，0 代表永不超时。
	ReadTimeout time.Duration

	// 底层库写入的超时时间，默认为 0，即永不超时。
	WriteTimeout time.Duration

	// 当 IdleTimeout 期间无请求时，服务器将关闭此连接。
	// 默认为 ReadTimeout，0 代表永不超时。
	IdleTimeout time.Duration

	// 是否将 /foo/ 重定向到 /foo，默认重定向。
	RedirectTrailingSlash bool

	// 将 /FOO 和 /..//FOO 重定向到 /foo，默认不重定向。
	RedirectFixedPath bool

	// 若启用，则方法不允许时路由器尝试查找
	HandleMethodNotAllowed bool

	// 用 url.RawPath 来查找参数。
	UseRawPath bool

	// 移除额外的斜线，以从URL中解析参数。
	RemoveExtraSlash bool

	// 是否不转义路径值，默认不转义。
	UnescapePathValues bool

	MaxRequestBodySize           int           // 最大请求正文字节数
	MaxKeepBodySize              int           // 最大保留正文字节数
	GetOnly                      bool          // 是否仅支持 GET 请求
	DisableKeepalive             bool          // 是否禁用长连接
	DisablePreParseMultipartForm bool          // 是否不预先解析多部分表单
	StreamRequestBody            bool          // 是否流式处理请求正文
	NoDefaultServerHeader        bool          // 是否不要默认的服务器名称标头
	DisablePrintRoute            bool          // 是否禁止打印路由
	Network                      string        // "tcp", "udp", "unix"(unix domain socket)，默认 "tcp"
	Addr                         string        // 监听地址，默认 ":8888"
	BasePath                     string        // 基本路径，默认 "/"
	ExitWaitTimeout              time.Duration // 优雅退出的等待时间，默认 5s。
	TLS                          *tls.Config
	ALPN                         bool  // ALPN 开关
	H2C                          bool  // H2C 开关
	ReadBufferSize               int   // 初始的读缓冲大小，默认 4KB。通常无需设置。
	Tracers                      []any // 一组链路跟踪器
	TraceLevel                   any   // 跟踪级别，默认 stats.LevelDetailed
	ListenConfig                 *net.ListenConfig

	// TransporterNewer 是传输器的自定义创建函数。
	TransporterNewer func(opt *Options) network.Transporter
	// AltTransporterNewer 是替补的传输器自定义创建函数。
	AltTransporterNewer func(opt *Options) network.Transporter

	// 在 netpoll 库中，OnAccept 是在接受连接之后且加到 epoll 之前调用的。OnConnect 是在加到 epoll 之后调用的。
	// 区别在于 OnConnect 能取数据，而 OnAccept 不能。例如想检查对端IP是否在黑名单中，可使用 OnAccept。
	//
	// 在 go/net 中，OnAccept 是在接受连接之后且建立 tls 连接之前调用的。建立 tls 连接后执行 OnConnect。
	OnAccept  func(conn net.Conn) context.Context
	OnConnect func(ctx context.Context, conn network.Conn) context.Context

	// 用于服务注册。
	Registry registry.Registry

	// 用于服务注册的信息。
	RegistryInfo *registry.Info

	// 启用 HTML 模板自动重载机制
	AutoReloadRender bool

	// HTML 模板自动重载时间间隔。
	// 默认为0，即根据文件变更事件立即重载。
	AutoReloadInterval time.Duration
}

// Apply 将指定的一组配置方法 opts 应用到配置项上。
func (o *Options) Apply(opts []Option) {
	for _, opt := range opts {
		opt.F(o)
	}
}

// NewOptions 创建配置项并应用指定的配置函数。
func NewOptions(opts []Option) *Options {
	options := &Options{
		KeepAliveTimeout:      defaultKeepAliveTimeout,
		ReadTimeout:           defaultReadTimeout,
		IdleTimeout:           defaultReadTimeout,
		RedirectTrailingSlash: true,
		UnescapePathValues:    true,
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
