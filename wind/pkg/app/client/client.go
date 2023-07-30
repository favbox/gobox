package client

import (
	"sync"

	"github.com/favbox/gosky/wind/internal/nocopy"
	"github.com/favbox/gosky/wind/pkg/common/config"
	"github.com/favbox/gosky/wind/pkg/protocol"
	"github.com/favbox/gosky/wind/pkg/protocol/client"
	"github.com/favbox/gosky/wind/pkg/protocol/suite"
)

// Client 实现 http 客户端。
//
// 禁止值拷贝 Client。可新建实例。
//
// Client 的方法是协程安全的。
type Client struct {
	noCopy nocopy.NoCopy

	options *config.ClientOptions

	// 指定一个函数，用于返回给定请求的代理。
	// 如果函数返回错误，则请求将中止。
	//
	// 代理类型由 URL scheme 决定。
	// 支持 "http" 和 "https"，若 schema 为空，则假定 "http"。
	//
	// 若 Proxy 为空或返回的 *URL 为空，则不使用代理。
	Proxy protocol.Proxy

	// 设置重试决策函数。若为空，则应用 client.DefaultRetryIf。
	RetryIfFunc client.RetryIfFunc

	clientFactory suite.ClientFactory

	mLock sync.Mutex
	m     map[string]client.HostClient
	ms    map[string]client.HostClient
	//mws            Middleware // TODO 待实现客户端中间件
	//lastMiddleware Middleware
}
