package http1

import (
	"crypto/tls"
	"time"

	"github.com/favbox/gosky/wind/internal/nocopy"
	"github.com/favbox/gosky/wind/pkg/app/client/retry"
	"github.com/favbox/gosky/wind/pkg/network"
	"github.com/favbox/gosky/wind/pkg/protocol/client"
)

type ClientOptions struct {
	// 客户端名称。用于 User-Agent 请求标头。
	Name string

	// 若在请求时排除 User-Agent 标头，则设为真。
	NoDefaultUserAgentHeader bool

	// 用于建立主机连接的回调
	//
	// 若未设置，则使用默认拨号器。
	Dialer network.Dialer

	// 建立主机连接的超时时间。
	//
	// 若为设置，则使用默认拨号超时时间。
	DialTimeout time.Duration

	// 若为真，则尝试同时连接 ipv4 和 ipv6 的主机地址。
	//
	// 该选项仅当使用默认 TCP 拨号器时可用，如 Dialer 未设置。
	//
	// 默认只连接到 ipv4 地址，因为 ipv6 在全球很多网络中处于故障状态。
	DialDualStack bool

	// 是否对主机连接使用 TLS（也叫 SSL 或 HTTPS），可选。
	TLSConfig *tls.Config

	// 最大主机连接数。
	//
	// 使用 HostClient 时，可通过 HostClient.SetMaxConns 更改此值。
	MaxConns int

	// Keep-alive 保活连接超过此时长会被关闭。
	//
	// 默认不限时长。
	MaxConnDuration time.Duration

	// 空闲连接超过此时长后会被关闭。
	//
	// 默认值为 DefaultMaxIdleConnDuration。
	MaxIdleConnDuration time.Duration

	// 完整响应的最大读取时长（包括正文）。
	//
	// 默认不限时长。
	ReadTimeout time.Duration

	// 完整请求的最大写入时长（包括正文）。
	//
	// 默认不限时长。
	WriteTimeout time.Duration

	// 响应主体的最大字节数。超限则客户端返回 errBodyTooLarge。
	//
	// 默认不限字节数大小。
	MaxResponseBodySize int

	// 若为真，则标头名称按原样传递，而无需规范化。
	//
	// 禁用标头名称的规范化，可能对代理其他需要区分标头大小写的客户端响应有用。
	//
	// 默认情况下，请求和响应的标头名称都要规范化，例如
	// 首字母和破折号后的首字母都转为大写，其余转为小写。
	// 示例：
	//
	//	* HOST -> Host
	//	* connect-type -> Content-Type
	//	* cONTENT-lenGTH -> Content-Length
	DisableHeaderNamesNormalizing bool

	// 若为真，则标头名称按原样传递，而无需规范化。
	//
	// 禁用路径的规范化，可能对代理期望保留原始路径的传入请求有用。
	DisablePathNormalizing bool

	// 等待一个空闲连接的最大时长。
	//
	// 默认不等待，立即返回 ErrNoFreeConns。
	MaxConnWaitTimeout time.Duration

	// 是否启用响应的主体流
	ResponseBodyStream bool

	// 与重试相关的所有配置
	RetryConfig *retry.Options

	RetryIfFunc client.RetryIfFunc

	// 观察主机客户端的状态
	StateObserve config.HostClientStateFunc

	// 观察间隔时长
	ObservationInterval time.Duration
}

type HostClient struct {
	noCopy nocopy.NoCopy

	*ClientOptions
}
