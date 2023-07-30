package http1

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"sync"
	"time"

	"github.com/favbox/gosky/wind/internal/bytestr"
	internalStats "github.com/favbox/gosky/wind/internal/stats"
	"github.com/favbox/gosky/wind/pkg/app"
	errs "github.com/favbox/gosky/wind/pkg/common/errors"
	"github.com/favbox/gosky/wind/pkg/common/tracer/stats"
	"github.com/favbox/gosky/wind/pkg/common/tracer/traceinfo"
	"github.com/favbox/gosky/wind/pkg/network"
	"github.com/favbox/gosky/wind/pkg/protocol"
	"github.com/favbox/gosky/wind/pkg/protocol/consts"
	"github.com/favbox/gosky/wind/pkg/protocol/http1/ext"
	"github.com/favbox/gosky/wind/pkg/protocol/http1/req"
	"github.com/favbox/gosky/wind/pkg/protocol/http1/resp"
	"github.com/favbox/gosky/wind/pkg/protocol/suite"
)

var (
	errIdleTimeout     = errs.New(errs.ErrIdleTimeout, errs.ErrorTypePublic, nil)
	errUnexpectedEOF   = errs.NewPublic(io.ErrUnexpectedEOF.Error() + " when reading request")
	errHijacked        = errs.New(errs.ErrHijacked, errs.ErrorTypePublic, nil)
	errShortConnection = errs.New(errs.ErrShortConnection, errs.ErrorTypePublic, "服务器将关闭连接")
)

// NewServer 创建新的 HTTP/1.1 服务器。
func NewServer() *Server {
	return &Server{
		eventStackPool: &sync.Pool{
			New: func() any {
				return &eventStack{}
			},
		},
	}
}

// Option 表示 HTTP/1.1 服务器选项。
type Option struct {
	StreamRequestBody            bool
	GetOnly                      bool
	DisablePreParseMultipartForm bool
	DisableKeepalive             bool
	NoDefaultServerHeader        bool
	MaxRequestBodySize           int
	IdleTimeout                  time.Duration
	ReadTimeout                  time.Duration
	ServerName                   []byte
	TLS                          *tls.Config
	EnableTrace                  bool
	ContinueHandler              func(header *protocol.RequestHeader) bool
	HijackConnHandle             func(c network.Conn, h app.HijackHandler)
}

// Server 表示 HTTP/1.1 服务器结构体。
//
// 实现 protocol.Server 协议服务器。
type Server struct {
	Option
	Core suite.Core

	// 存储 TraceInfo 处理函数的堆栈池
	eventStackPool *sync.Pool
}

func (s Server) Serve(c context.Context, conn network.Conn) (err error) {
	var (
		zr network.Reader
		zw network.Writer

		serverName      []byte
		isHTTP11        bool
		connectionClose bool

		continueReadingRequest = true

		hijackHandler app.HijackHandler

		// HTTP1 路径
		// 1. 获取请求上下文
		// 2. 准备它
		// 3. 处理它
		// 4. 重置和回收
		ctx = s.Core.GetCtxPool().Get().(*app.RequestContext)

		traceCtl        = s.Core.GetTracer()
		eventsToTrigger *eventStack

		// 使用新变量保存标准上下文，以免修改初始上下文。
		cc = c
	)

	if s.EnableTrace {
		eventsToTrigger = s.eventStackPool.Get().(*eventStack)
	}

	defer func() {
		if s.EnableTrace {
			if err != nil && !errors.Is(err, errs.ErrIdleTimeout) && !errors.Is(err, errs.ErrHijacked) {
				ctx.GetTraceInfo().Stats().SetError(err)
			}
			// 如果出现错误，我们需要触发所有事件
			if eventsToTrigger != nil {
				for last := eventsToTrigger.pop(); last != nil; last = eventsToTrigger.pop() {
					last(ctx.GetTraceInfo(), err)
				}
				s.eventStackPool.Put(eventsToTrigger)
			}

			traceCtl.DoFinish(cc, ctx, err)
		}

		// Hijack 可能已经释放并关闭连接了
		if zr != nil && !errors.Is(err, errs.ErrHijacked) {
			_ = zr.Release()
			zr = nil
		}
		ctx.Reset()
		s.Core.GetCtxPool().Put(ctx)
	}()

	// TODO  HTML 渲染器
	//ctx.HTMLRender = s.HTMLRender
	ctx.SetConn(conn)
	ctx.Request.SetIsTLS(s.TLS != nil)
	ctx.SetEnableTrace(s.EnableTrace)

	if !s.NoDefaultServerHeader {
		serverName = s.ServerName
	}

	connRequestNum := uint64(0)

	for {
		connRequestNum++

		if zr == nil {
			zr = ctx.GetReader()
		}

		// 若为保活链接，则尝试读取在空闲时间内读取前面几个字节。
		if connRequestNum > 1 {
			_ = ctx.GetConn().SetReadTimeout(s.IdleTimeout)

			_, err = zr.Peek(4)
			// 这不是第一个请求，我们还未读取新请求的第一个字节。
			// 这意味着它只是一个保活连接的关闭，要么是远端关闭了它，
			// 要么是由于我们这边的读取超时。无论哪种方式，只要关闭连接，
			// 都不会返回任何错误响应。
			if err != nil {
				err = errIdleTimeout
				return
			}

			// 重置后续请求的真实读取超时时长
			_ = ctx.GetConn().SetReadTimeout(s.ReadTimeout)
		}

		// 跟踪器记录请求开始和结束信息。
		if s.EnableTrace {
			traceCtl.DoStart(c, ctx)
			internalStats.Record(ctx.GetTraceInfo(), stats.ReadHeaderStart, err)
			eventsToTrigger.push(func(ti traceinfo.TraceInfo, err error) {
				internalStats.Record(ti, stats.ReadHeaderFinish, err)
			})
		}

		// 读取标头
		if err = req.ReadHeader(&ctx.Request.Header, zr); err == nil {
			if s.EnableTrace {
				// 读取标头完成
				if last := eventsToTrigger.pop(); last != nil {
					last(ctx.GetTraceInfo(), err)
				}
				internalStats.Record(ctx.GetTraceInfo(), stats.ReadBodyStart, err)
				eventsToTrigger.push(func(ti traceinfo.TraceInfo, err error) {
					internalStats.Record(ti, stats.ReadBodyFinish, err)
				})
			}
			// 读取正文
			if s.StreamRequestBody {
				err = req.ReadBodyStream(&ctx.Request, zr, s.MaxRequestBodySize, s.GetOnly, !s.DisablePreParseMultipartForm)
			} else {
				err = req.ReadLimitBody(&ctx.Request, zr, s.MaxRequestBodySize, s.GetOnly, !s.DisablePreParseMultipartForm)
			}
		}

		// 跟踪器设置接收内容的大小
		if s.EnableTrace {
			if ctx.Request.Header.ContentLength() >= 0 {
				ctx.GetTraceInfo().Stats().SetRecvSize(len(ctx.Request.Header.RawHeaders()) + ctx.Request.Header.ContentLength())
			} else {
				ctx.GetTraceInfo().Stats().SetRecvSize(0)
			}
			// 读取正文结束
			if last := eventsToTrigger.pop(); last != nil {
				last(ctx.GetTraceInfo(), err)
			}
		}

		// 读取正文出错
		if err != nil {
			if errors.Is(err, errs.ErrNothingRead) {
				return nil
			}

			if err == io.EOF {
				return errUnexpectedEOF
			}

			writeErrorResponse(zw, ctx, serverName, err)
			return
		}

		// 'Except: 100-continue' 请求处理。
		// 详见 https://www.w3.org/Protocols/rfc2616/rfc2616-sec8.html#sec8.2.3
		if ctx.Request.MayContinue() {
			// 允许拒绝读取后续的请求正文
			if s.ContinueHandler != nil {
				if continueReadingRequest = s.ContinueHandler(&ctx.Request.Header); !continueReadingRequest {
					ctx.SetStatusCode(consts.StatusExpectationFailed)
				}
			}

			if continueReadingRequest {
				zw = ctx.GetWriter()
				// 发送 'HTTP/1.1 100 Continue' 响应。
				_, err = zw.WriteBinary(bytestr.StrResponseContinue)
				if err != nil {
					return
				}
				err = zw.Flush()
				if err != nil {
					return
				}

				// 读取正文。
				if zr == nil {
					zr = ctx.GetReader()
				}
				if s.StreamRequestBody {
					err = req.ContinueReadBodyStream(&ctx.Request, zr, s.MaxRequestBodySize, !s.DisablePreParseMultipartForm)
				} else {
					err = req.ContinueReadBody(&ctx.Request, zr, s.MaxRequestBodySize, !s.DisablePreParseMultipartForm)
				}
				if err != nil {
					writeErrorResponse(zw, ctx, serverName, err)
					return
				}
			}
		}

		connectionClose = s.DisableKeepalive || ctx.Request.Header.ConnectionClose()
		isHTTP11 = ctx.Request.Header.IsHTTP11()

		// 设置服务器名称。
		if serverName != nil {
			ctx.Response.Header.SetServerBytes(serverName)
		}
		if s.EnableTrace {
			internalStats.Record(ctx.GetTraceInfo(), stats.ServerHandleStart, err)
			eventsToTrigger.push(func(ti traceinfo.TraceInfo, err error) {
				internalStats.Record(ti, stats.ServerHandleFinish, err)
			})
		}

		// 处理请求。
		//
		// 注意：所有的中间件和业务处理器都将在此执行。
		// 此时，请求已被解析，路由也已匹配。
		s.Core.ServeHTTP(cc, ctx)
		if s.EnableTrace {
			// 应用层处理结束
			if last := eventsToTrigger.pop(); last != nil {
				last(ctx.GetTraceInfo(), err)
			}
		}

		// 退出检查
		if !s.Core.IsRunning() {
			connectionClose = true
		}

		if !ctx.IsGet() && ctx.IsHead() {
			ctx.Response.SkipBody = true
		}

		hijackHandler = ctx.GetHijackHandler()
		ctx.SetHijackHandler(nil)

		connectionClose = connectionClose || ctx.Response.ConnectionClose()
		if connectionClose {
			ctx.Response.Header.SetCanonical(bytestr.StrConnection, bytestr.StrClose)
		} else if !isHTTP11 {
			ctx.Response.Header.SetCanonical(bytestr.StrConnection, bytestr.StrKeepAlive)
		}

		// 写入响应
		if zw == nil {
			zw = ctx.GetWriter()
		}
		if s.EnableTrace {
			internalStats.Record(ctx.GetTraceInfo(), stats.WriteStart, err)
			eventsToTrigger.push(func(ti traceinfo.TraceInfo, err error) {
				internalStats.Record(ti, stats.WriteFinish, err)
			})
		}
		if err = writeResponse(ctx, zw); err != nil {
			return
		}

		// 跟踪器设置发送大小
		if s.EnableTrace {
			if ctx.Response.Header.ContentLength() > 0 {
				ctx.GetTraceInfo().Stats().SetSendSize(ctx.Response.Header.GetHeaderLength() + ctx.Response.Header.ContentLength())
			} else {
				ctx.GetTraceInfo().Stats().SetSendSize(0)
			}
		}

		// 在刷新前释放 zeroCopyReader 以防数据竞赛
		if zr != nil {
			zr.Release()
			zr = nil
		}
		// 刷新响应。
		if err = zw.Flush(); err != nil {
			return
		}
		if s.EnableTrace {
			// 写入完成
			if last := eventsToTrigger.pop(); last != nil {
				last(ctx.GetTraceInfo(), err)
			}
		}

		// 释放请求正文流
		if ctx.Request.IsBodyStream() {
			err = ext.ReleaseBodyStream(ctx.RequestBodyStream())
			if err != nil {
				return
			}
		}

		// 处理劫持连接
		if hijackHandler != nil {
			// 劫持连接自己处理超时
			err = ctx.GetConn().SetReadTimeout(0)
			if err != nil {
				return
			}

			// 劫持并阻塞连接，支持 hijackHandler 返回
			s.HijackConnHandle(ctx.GetConn(), hijackHandler)
			err = errHijacked
			return
		}

		// 跟踪器处理完成情况
		if s.EnableTrace {
			traceCtl.DoFinish(cc, ctx, err)
		}

		// 返回待关闭指示
		if connectionClose {
			return errShortConnection
		}

		// 返回网络层进行处罚。
		// 目前，只有 netpoll 的网络模式由此特性。
		if s.IdleTimeout == 0 {
			return
		}

		ctx.ResetWithoutConn()
	}
}

func defaultErrorHandler(ctx *app.RequestContext, err error) {
	if netErr, ok := err.(*net.OpError); ok && netErr.Timeout() {
		ctx.AbortWithMsg("请求超时", consts.StatusRequestTimeout)
	} else if errors.Is(err, errs.ErrBodyTooLarge) {
		ctx.AbortWithMsg("请求实体太大", consts.StatusRequestEntityTooLarge)
	} else {
		ctx.AbortWithMsg("解析请求时出错", consts.StatusBadRequest)
	}
}

func writeErrorResponse(zw network.Writer, ctx *app.RequestContext, serverName []byte, err error) network.Writer {
	errorHandler := defaultErrorHandler

	errorHandler(ctx, err)

	if serverName != nil {
		ctx.Response.Header.SetServerBytes(serverName)
	}
	ctx.SetConnectionClose()
	if zw == nil {
		zw = ctx.GetWriter()
	}
	writeResponse(ctx, zw)
	zw.Flush()
	return zw
}

func writeResponse(ctx *app.RequestContext, w network.Writer) error {
	// 若连接已被劫持，则跳过默认响应的写入逻辑
	if ctx.Response.GetHijackWriter() != nil {
		return ctx.Response.GetHijackWriter().Finalize()
	}

	err := resp.Write(&ctx.Response, w)
	if err != nil {
		return err
	}

	return err
}

type eventStack []func(ti traceinfo.TraceInfo, err error)

func (e *eventStack) isEmpty() bool {
	return len(*e) == 0
}

// 追加一个跟踪信息回调函数。
func (e *eventStack) push(f func(ti traceinfo.TraceInfo, err error)) {
	*e = append(*e, f)
}

// 弹出最后一个跟踪信息回调函数。
func (e *eventStack) pop() func(ti traceinfo.TraceInfo, err error) {
	if e.isEmpty() {
		return nil
	}
	last := (*e)[len(*e)-1]
	*e = (*e)[:len(*e)-1]
	return last
}
