package http1

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	errs "github.com/favbox/gosky/wind/pkg/common/errors"
	"github.com/favbox/gosky/wind/pkg/common/test/mock"
	"github.com/favbox/gosky/wind/pkg/network"
	"github.com/favbox/gosky/wind/pkg/protocol"
	"github.com/favbox/gosky/wind/pkg/protocol/consts"
)

type mockDialer struct {
	customDialConn func(network, addr string) (network.Conn, error)
}

func (m *mockDialer) DialConnection(network, address string, timeout time.Duration, tlsConfig *tls.Config) (conn network.Conn, err error) {
	return m.customDialConn(network, address)
}

func (m *mockDialer) DialTimeout(network, address string, timeout time.Duration, tlsConfig *tls.Config) (conn net.Conn, err error) {
	return nil, nil
}

func (m *mockDialer) AddTLS(conn network.Conn, tlsConfig *tls.Config) (network.Conn, error) {
	return nil, nil
}

type slowDialer struct {
	*mockDialer
}

func (s *slowDialer) DialConnection(network, address string, timeout time.Duration, tlsConfig *tls.Config) (conn network.Conn, err error) {
	time.Sleep(timeout)
	return nil, errs.ErrDialTimeout
}

func TestHostClient_MaxConnWaitTimeoutWithEarlierDeadline(t *testing.T) {
	var (
		emptyBodyCount uint8
		wg             sync.WaitGroup
		// 使截止时间早于连接超时时长
		timeout = 10 * time.Millisecond
	)

	c := &HostClient{
		ClientOptions: &ClientOptions{
			Dialer: newSlowConnDialer(func(network, addr string) (network.Conn, error) {
				return mock.SlowReadDialer(addr)
			}),
			MaxConns:           1,
			MaxConnWaitTimeout: 50 * time.Millisecond,
		},
		Addr: "foobar",
	}

	var errTimeoutCount uint32
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			req := protocol.AcquireRequest()
			req.SetRequestURI("http://foobar/baz")
			req.Header.SetMethod(consts.MethodPost)
			req.SetBodyString("bar")
			resp := protocol.AcquireResponse()

			if err := c.DoDeadline(context.Background(), req, resp, time.Now().Add(timeout)); err != nil {
				if !errors.Is(err, errs.ErrTimeout) {
					t.Errorf("异常错误：%s。期待：%s", err, errs.ErrTimeout)
				}
				atomic.AddUint32(&errTimeoutCount, 1)
			} else {
				if resp.StatusCode() != consts.StatusOK {
					t.Errorf("异常的状态码 %d。期待 %d", resp.StatusCode(), consts.StatusOK)
				}

				body := resp.Body()
				if string(body) != "foo" {
					t.Errorf("异常的正文 %q。期待 %q", body, "abcd")
				}
			}
		}()
	}
	wg.Wait()

	c.connsLock.Lock()
	for {
		w := c.connsWait.popFront()
		if w == nil {
			break
		}
		w.mu.Lock()
		if w.err != nil && !errors.Is(w.err, errs.ErrNoFreeConns) {
			t.Errorf("异常错误：%s。期待：%s", w.err, errs.ErrNoFreeConns)
		}
		w.mu.Unlock()
	}
	c.connsLock.Unlock()
	if errTimeoutCount == 0 {
		t.Errorf("异常的 errTimeoutCount: %d. 期待 > 0", errTimeoutCount)
	}

	if emptyBodyCount > 0 {
		t.Fatalf("至少有一个请求体为空")
	}
}

func newSlowConnDialer(dialer func(network string, addr string) (network.Conn, error)) network.Dialer {
	return &mockDialer{customDialConn: dialer}
}
