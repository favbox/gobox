package resp

import (
	"errors"
	"io"
	"runtime"
	"sync"

	errs "github.com/favbox/gosky/wind/pkg/common/errors"
	"github.com/favbox/gosky/wind/pkg/common/hlog"
	"github.com/favbox/gosky/wind/pkg/network"
	"github.com/favbox/gosky/wind/pkg/protocol"
	"github.com/favbox/gosky/wind/pkg/protocol/consts"
	"github.com/favbox/gosky/wind/pkg/protocol/http1/ext"
)

// ReadBodyStream 流式读取 r 到响应 resp。
func ReadBodyStream(resp *protocol.Response, r network.Reader, maxBodySize int, closeCallback func(shouldClose bool) error) error {
	resp.ResetBody()
	err := ReadHeader(&resp.Header, r)
	if err != nil {
		return err
	}

	if resp.Header.StatusCode() == consts.StatusContinue {
		// 读取下一个响应，根据 http://www.w3.org/Protocols/rfc2616/rfc2616-sec8.html
		if err = ReadHeader(&resp.Header, r); err != nil {
			return err
		}
	}

	if resp.MustSkipBody() {
		return nil
	}

	bodyBuf := resp.BodyBuffer()
	bodyBuf.Reset()
	bodyBuf.B, err = ext.ReadBodyWithStreaming(r, resp.Header.ContentLength(), maxBodySize, bodyBuf.B)
	if err != nil {
		if errors.Is(err, errs.ErrBodyTooLarge) {
			bodyStream := ext.AcquireBodyStream(bodyBuf, r, resp.Header.Trailer(), resp.Header.ContentLength())
			resp.ConstructBodyStream(bodyBuf, convertClientRespStream(bodyStream, closeCallback))
			return nil
		}

		if errors.Is(err, errs.ErrChunkedStream) {
			bodyStream := ext.AcquireBodyStream(bodyBuf, r, resp.Header.Trailer(), -1)
			resp.ConstructBodyStream(bodyBuf, convertClientRespStream(bodyStream, closeCallback))
			return nil
		}

		resp.Reset()
		return err
	}

	bodyStream := ext.AcquireBodyStream(bodyBuf, r, resp.Header.Trailer(), resp.Header.ContentLength())
	resp.ConstructBodyStream(bodyBuf, convertClientRespStream(bodyStream, closeCallback))
	return nil
}

// Read 读取 r 到请求 req（包括正文）。
//
// 若 r 已关闭则返回 io.EOF。
func Read(resp *protocol.Response, r network.Reader) error {
	return ReadHeaderAndLimitBody(resp, r, 0)
}

// ReadHeaderAndLimitBody 读取 r 到请求 req，限定正文大小。
//
// 若 maxBodySize > 0 且正文大小超此限制，则 ErrBodyTooLarge 将被返回。
//
// 若 r 已关闭则返回 io.EOF。
func ReadHeaderAndLimitBody(resp *protocol.Response, r network.Reader, maxBodySize int) error {
	resp.ResetBody()
	err := ReadHeader(&resp.Header, r)
	if err != nil {
		return err
	}
	if resp.Header.StatusCode() == consts.StatusContinue {
		// 读取下一个响应，根据 http://www.w3.org/Protocols/rfc2616/rfc2616-sec8.html
		if err = ReadHeader(&resp.Header, r); err != nil {
			return err
		}
	}

	if !resp.MustSkipBody() {
		bodyBuf := resp.BodyBuffer()
		bodyBuf.Reset()
		bodyBuf.B, err = ext.ReadBody(r, resp.Header.ContentLength(), maxBodySize, bodyBuf.B)
		if err != nil {
			return err
		}
		if resp.Header.ContentLength() == -1 {
			err = ext.ReadTrailer(resp.Header.Trailer(), r)
			if err != nil && err != io.EOF {
				return err
			}
		}
		resp.Header.SetContentLength(len(bodyBuf.B))
	}

	return nil
}

// Write 将响应 resp 写入网络编写器 w。
//
// Write 出于性能原因不会将响应冲刷 到 w。
func Write(resp *protocol.Response, w network.Writer) error {
	// TODO
}

var clientRespStreamPool = sync.Pool{
	New: func() any {
		return &clientRespStream{}
	},
}

// 池化管理的客户端响应流。
type clientRespStream struct {
	r             io.Reader
	closeCallback func(shouldClose bool) error
}

func (c *clientRespStream) Read(p []byte) (n int, err error) {
	return c.r.Read(p)
}

func (c *clientRespStream) Close() error {
	runtime.SetFinalizer(c, nil)
	// 如果释放时发生错误，则连接可能处于异常状态。
	// 在回调中关闭它，以避免其他意外问题。
	err := ext.ReleaseBodyStream(c.r)
	shouldClose := false
	if err != nil {
		shouldClose = true
		hlog.Warnf("连接即将关闭而非回收，因为在正文流释放过程中发生了错误：%s", err.Error())
	}
	if c.closeCallback != nil {
		err = c.closeCallback(shouldClose)
	}
	c.reset()
	return err
}

func (c *clientRespStream) reset() {
	c.closeCallback = nil
	c.r = nil
	clientRespStreamPool.Put(c)
}

func convertClientRespStream(bs io.Reader, fn func(shouldClose bool) error) *clientRespStream {
	clientStream := clientRespStreamPool.Get().(*clientRespStream)
	clientStream.r = bs
	clientStream.closeCallback = fn
	runtime.SetFinalizer(clientStream, (*clientRespStream).Close)
	return clientStream
}
