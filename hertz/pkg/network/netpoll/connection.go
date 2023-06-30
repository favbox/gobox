package netpoll

import (
	"errors"
	"io"
	"strings"
	"syscall"

	"github.com/cloudwego/netpoll"
	errs "github.com/favbox/gobox/hertz/pkg/common/errors"
	"github.com/favbox/gobox/hertz/pkg/common/hlog"
	"github.com/favbox/gobox/hertz/pkg/network"
)

type Conn struct {
	network.Conn
}

func (c *Conn) ToHertzError(err error) error {
	if errors.Is(err, netpoll.ErrConnClosed) || errors.Is(err, syscall.EPIPE) {
		return errs.ErrConnectionClosed
	}
	return err
}

func (c *Conn) Peek(n int) (b []byte, err error) {
	b, err = c.Conn.Peek(n)
	err = normalizeErr(err)
	return
}

func (c *Conn) Read(p []byte) (int, error) {
	n, err := c.Conn.Read(p)
	err = normalizeErr(err)
	return n, err
}

func (c *Conn) Skip(n int) error {
	return c.Conn.Skip(n)
}

func (c *Conn) Release() error {
	return c.Conn.Release()
}

func (c *Conn) Len() int {
	return c.Conn.Len()
}

func (c *Conn) ReadByte() (b byte, err error) {
	b, err = c.Conn.ReadByte()
	err = normalizeErr(err)
	return
}

func (c *Conn) ReadBinary(n int) (b []byte, err error) {
	b, err = c.Conn.ReadBinary(n)
	err = normalizeErr(err)
	return
}

func (c *Conn) Malloc(n int) (buf []byte, err error) {
	return c.Conn.Malloc(n)
}

func (c *Conn) WriteBinary(b []byte) (n int, err error) {
	return c.Conn.WriteBinary(b)
}

func (c *Conn) Flush() error {
	return c.Conn.Flush()
}

// HandleSpecificError 判断特定错误是否需要忽略。
func (c *Conn) HandleSpecificError(err error, remoteIP string) (needIgnore bool) {
	// 需要忽略错误
	if errors.Is(err, netpoll.ErrConnClosed) || errors.Is(err, syscall.EPIPE) || errors.Is(err, syscall.ECONNRESET) {
		// 忽略因连接被关闭或重置产生的 flush 错误
		if strings.Contains(err.Error(), "when flush") {
			return true
		}
		hlog.SystemLogger().Debugf("Netpoll error=%s, remoteAddr=%s", err.Error(), remoteIP)
		return true
	}

	// 其他为不可忽略的错误
	return false
}

func normalizeErr(err error) error {
	if errors.Is(err, netpoll.ErrEOF) {
		return io.EOF
	}

	return err
}

// 将 netpoll 连接转为 hertz HTTP 连接
func newConn(c netpoll.Connection) network.Conn {
	return &Conn{Conn: c.(network.Conn)}
}
