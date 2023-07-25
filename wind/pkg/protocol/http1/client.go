package http1

import (
	"bytes"
	"crypto/tls"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/favbox/gosky/wind/internal/bytesconv"
	"github.com/favbox/gosky/wind/internal/bytestr"
	"github.com/favbox/gosky/wind/internal/nocopy"
	"github.com/favbox/gosky/wind/pkg/app/client/retry"
	"github.com/favbox/gosky/wind/pkg/common/config"
	errs "github.com/favbox/gosky/wind/pkg/common/errors"
	"github.com/favbox/gosky/wind/pkg/common/hlog"
	"github.com/favbox/gosky/wind/pkg/network"
	"github.com/favbox/gosky/wind/pkg/network/dialer"
	"github.com/favbox/gosky/wind/pkg/protocol"
	"github.com/favbox/gosky/wind/pkg/protocol/client"
)

var (
	startTimeUnix = time.Now().Unix()

	clientConnPool sync.Pool

	errTimeout          = errs.New(errs.ErrTimeout, errs.ErrorTypePublic, "host client")
	errConnectionClosed = errs.NewPublic("服务器在返回响应前已关闭了连接。要确保服务器在关闭连接前返回 'Connection: close' 响应标头")
)

type clientConn struct {
	c network.Conn

	createdTime time.Time
	lastUseTime time.Time
}

type wantConn struct {
	ready chan struct{}
	mu    sync.Mutex // 保护 conn, err, close(ready)
	conn  *clientConn
	err   error
}

// 尝试传递 conn, err 给当前 w，并汇报是否成功。
func (w *wantConn) tryDeliver(conn *clientConn, err error) bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.conn != nil || w.err != nil {
		return false
	}
	w.conn = conn
	w.err = err
	if w.conn == nil && w.err == nil {
		panic("wind: 内部错误: 滥用 tryDeliver")
	}
	close(w.ready)
	return true
}

// 返回该等待连接是否还在等待答案（连接或错误）
func (w *wantConn) waiting() bool {
	select {
	case <-w.ready:
		return false
	default:
		return true
	}
}

// 标记 w 不在等待结果（如：取消了）
// 如果连接已被传递，则调用 HostClient.releaseConn 进行释放。
func (w *wantConn) close(c *HostClient, err error) {
	w.mu.Lock()
	if w.conn == nil && w.err == nil {
		close(w.ready) // 在未来传递中发现不当行为
	}

	conn := w.conn
	w.conn = nil
	w.err = err
	w.mu.Unlock()

	if conn != nil {
		c.releaseConn(conn)
	}
}

type wantConnQueue struct {
	head    []*wantConn
	headPos int
	tail    []*wantConn
}

// 返回队列中的连接数。
func (q *wantConnQueue) len() int {
	return len(q.head) - q.headPos + len(q.tail)
}

// 返回队首的等待连接但不删除它。
func (q *wantConnQueue) peekFront() *wantConn {
	if q.headPos < len(q.head) {
		return q.head[q.headPos]
	}
	if len(q.tail) > 0 {
		return q.tail[0]
	}
	return nil
}

// 移除并返回队首的等待连接。
func (q *wantConnQueue) popFront() *wantConn {
	if q.headPos >= len(q.head) {
		if len(q.tail) == 0 {
			return nil
		}
		// 把尾巴当做新的头，并清掉尾巴。
		q.head, q.headPos, q.tail = q.tail, 0, q.head[:0]
	}

	w := q.head[q.headPos]
	q.head[q.headPos] = nil
	q.headPos++
	return w
}

// 清掉队首不再等待的连接，并返回其是否已被弹出。
func (q *wantConnQueue) clearFront() (cleaned bool) {
	for {
		w := q.peekFront()
		if w == nil || w.waiting() {
			return cleaned
		}
		q.popFront()
		cleaned = true
	}
}

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

	// 双重拨号，若为真，则尝试同时连接 ipv4 和 ipv6 的主机地址。
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

// HostClient 平衡 Addr 中列举的主机之间的 http 请求。
//
// HostClient 可用于在多个上游主机之间平衡负载。
// 虽然 Addr 的多个主机可平衡他们之间的负载，但最好使用专用的 LBClient。
//
// 禁止拷贝 HostClient 实例。可以创建新实例。
//
// 并发协程的调用是安全的。
type HostClient struct {
	noCopy nocopy.NoCopy

	*ClientOptions

	// 逗号分隔的上游 HTTP 服务器主机地址列表，以循环方式传递给 Dialer。
	//
	// 如果使用默认拨号程序，则每个地址都可能包含端口。
	// 例如：
	//
	//	- foobar.com:80
	//	- foobar.com:443
	//	- foobar.com:8080
	Addr     string
	IsTLS    bool
	ProxyURI *protocol.URI

	clientName  atomic.Value
	lastUseTime uint32

	connsLock  sync.Mutex
	connsCount int
	conns      []*clientConn
	connsWait  *wantConnQueue

	addrsLock sync.Mutex
	addrs     []string
	addrIdx   uint32

	tlsConfigMap     map[string]*tls.Config
	tlsConfigMapLock sync.Mutex

	pendingRequests int32

	connsCleanerRun bool

	closed chan struct{}
}

func (c *HostClient) Close() error {
	close(c.closed)
	return nil
}

// CloseIdleConnections 关闭所有之前请求建立而当前空闲却保持 "keep-alive" 状态的连接。
// 不会中断当前正在使用的连接。
func (c *HostClient) CloseIdleConnections() {
	c.connsLock.Lock()
	scratch := append([]*clientConn{}, c.conns...)
	for i := range c.conns {
		c.conns[i] = nil
	}
	c.conns = c.conns[:0]
	c.connsLock.Unlock()

	for _, cc := range scratch {
		c.closeConn(cc)
	}
}

func (c *HostClient) closeConn(cc *clientConn) {
	c.decConnsCount()
	cc.c.Close()
	releaseClientConn(cc)
}

func (c *HostClient) decConnsCount() {
	if c.MaxConnWaitTimeout <= 0 {
		c.connsLock.Lock()
		c.connsCount--
		c.connsLock.Unlock()
		return
	}

	c.connsLock.Lock()
	defer c.connsLock.Unlock()
	dialed := false
	if q := c.connsWait; q != nil && q.len() > 0 {
		for q.len() > 0 {
			w := q.popFront()
			if w.waiting() {
				go c.dialConnFor(w)
				dialed = true
				break
			}
		}
	}
	if !dialed {
		c.connsCount--
	}
}

func (c *HostClient) releaseConn(cc *clientConn) {
	cc.lastUseTime = time.Now()
	if c.MaxConnWaitTimeout <= 0 {
		c.connsLock.Lock()
		c.conns = append(c.conns, cc)
		c.connsLock.Unlock()
		return
	}

	// 尝试将空闲连接传递给正在等待的连接
	c.connsLock.Lock()
	defer c.connsLock.Unlock()
	delivered := false
	if q := c.connsWait; q != nil && q.len() > 0 {
		for q.len() > 0 {
			w := q.popFront()
			if w.waiting() {
				delivered = w.tryDeliver(cc, nil)
				break
			}
		}
	}
	// 传递失败则追加
	if !delivered {
		c.conns = append(c.conns, cc)
	}
}

func (c *HostClient) dialConnFor(w *wantConn) {
	conn, err := c.dialHostHard(c.DialTimeout)
	if err != nil {
		w.tryDeliver(nil, err)
		c.decConnsCount()
		return
	}

	cc := acquireClientConn(conn)
	delivered := w.tryDeliver(cc, nil)
	if !delivered {
		// 未送达，返回空闲连接
		c.releaseConn(cc)
	}
}

func (c *HostClient) dialHostHard(dialTimeout time.Duration) (conn network.Conn, err error) {
	// 在放弃之前尝试拨打所有可用的主机

	c.addrsLock.Lock()
	n := len(c.addrs)
	c.addrsLock.Unlock()

	if n == 0 {
		// 看起来 c.addrs 尚未初始化
		n = 1
	}

	deadline := time.Now().Add(dialTimeout)
	for n > 0 {
		addr := c.nextAddr()
		tlsConfig := c.cachedTLSConfig(addr)
		conn, err = dialAddr(addr, c.Dialer, c.DialDualStack, tlsConfig, dialTimeout, c.ProxyURI, c.IsTLS)
		if err == nil {
			return conn, nil
		}
		if time.Since(deadline) >= 0 {
			break
		}
		n--
	}
	return nil, err
}

func dialAddr(addr string, dial network.Dialer, dialDualStack bool, tlsConfig *tls.Config, timeout time.Duration, proxyURI *protocol.URI, isTLS bool) (network.Conn, error) {
	var conn network.Conn
	var err error
	if dial == nil {
		hlog.SystemLogger().Warn("HostClient: 未指定拨号器，尝试使用默认拨号器")
		dial = dialer.DefaultDialer()
	}
	dialFunc := dial.DialConnection

	// 地址已有端口号，此处无需操作
	if proxyURI != nil {
		// 先用 tcp 连接，代理将向其添加 TLS
		conn, err = dialFunc("tcp", string(proxyURI.Host()), timeout, nil)
	} else {
		conn, err = dialFunc("tcp", addr, timeout, tlsConfig)
	}

	if err != nil {
		return nil, err
	}
	if conn == nil {
		panic("BUG: dial.DialConnection 返回了 (nil, nil)")
	}

	if proxyURI != nil {
		conn, err = proxy.SetupProxy(conn, addr, proxyURI, tlsConfig, isTLS, dial)
	}
}

func (c *HostClient) nextAddr() string {
	c.addrsLock.Lock()
	if c.addrs == nil {
		c.addrs = strings.Split(c.Addr, ",")
	}
	addr := c.addrs[0]
	if len(c.addrs) > 1 {
		addr = c.addrs[c.addrIdx%uint32(len(c.addrs))]
		c.addrIdx++
	}
	c.addrsLock.Unlock()
	return addr
}

func (c *HostClient) cachedTLSConfig(addr string) *tls.Config {
	var cfgAddr string
	if c.ProxyURI != nil && bytes.Equal(c.ProxyURI.Scheme(), bytestr.StrHTTPS) {
		cfgAddr = bytesconv.B2s(c.ProxyURI.Host())
	}

	if c.IsTLS && cfgAddr == "" {
		cfgAddr = addr
	}

	if cfgAddr == "" {
		return nil
	}

	c.tlsConfigMapLock.Lock()
	if c.tlsConfigMap == nil {
		c.tlsConfigMap = make(map[string]*tls.Config)
	}
	cfg := c.tlsConfigMap[cfgAddr]
	if cfg == nil {
		cfg = newClientTLSConfig(c.TLSConfig, cfgAddr)
		c.tlsConfigMap[cfgAddr] = cfg
	}
	c.tlsConfigMapLock.Unlock()

	return cfg
}

func newClientTLSConfig(c *tls.Config, addr string) *tls.Config {
	if c == nil {
		c = &tls.Config{}
	} else {
		c = c.Clone()
	}

	if c.ClientSessionCache == nil {
		c.ClientSessionCache = tls.NewLRUClientSessionCache(0)
	}

	if len(c.ServerName) == 0 {
		serverName := tlsServerName(addr)
		if serverName == "*" {
			c.InsecureSkipVerify = true
		} else {
			c.ServerName = serverName
		}
	}

	return c
}

func tlsServerName(addr string) string {
	if !strings.Contains(addr, ":") {
		return addr
	}
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return "*"
	}
	return host
}

func acquireClientConn(conn network.Conn) *clientConn {
	v := clientConnPool.Get()
	if v == nil {
		v = &clientConn{}
	}
	cc := v.(*clientConn)
	cc.c = conn
	cc.createdTime = time.Now()
	return cc
}

func releaseClientConn(cc *clientConn) {
	// 重设所有字段。
	*cc = clientConn{}
	clientConnPool.Put(cc)
}
