package http1

import (
	"context"
	"errors"
	"sync"
	"testing"

	internalStats "github.com/favbox/gosky/wind/internal/stats"
	"github.com/favbox/gosky/wind/pkg/app"
	errs "github.com/favbox/gosky/wind/pkg/common/errors"
	"github.com/favbox/gosky/wind/pkg/common/test/assert"
	"github.com/favbox/gosky/wind/pkg/common/test/mock"
	"github.com/favbox/gosky/wind/pkg/common/tracer"
	"github.com/favbox/gosky/wind/pkg/common/tracer/stats"
	"github.com/favbox/gosky/wind/pkg/common/tracer/traceinfo"
	"github.com/favbox/gosky/wind/pkg/network"
)

var pool = &sync.Pool{
	New: func() any {
		return &eventStack{}
	},
}

type mockCore struct {
	ctxPool     *sync.Pool
	controller  tracer.Controller
	mockHandler func(c context.Context, ctx *app.RequestContext)
	isRunning   bool
}

func (m *mockCore) IsRunning() bool {
	return m.isRunning
}

func (m *mockCore) GetCtxPool() *sync.Pool {
	return m.ctxPool
}

func (m *mockCore) ServeHTTP(c context.Context, ctx *app.RequestContext) {
	if m.mockHandler != nil {
		m.mockHandler(c, ctx)
	}
}

func (m *mockCore) GetTracer() tracer.Controller {
	return m.controller
}

type mockTraceInfo struct {
	traceinfo.TraceInfo
}

func (m *mockTraceInfo) Reset() {}

type mockErrorWriter struct {
	network.Conn
}

func (errWriter *mockErrorWriter) Flush() error {
	return errors.New("error")
}

func TestTraceEventCompleted(t *testing.T) {
	server := &Server{}
	server.eventStackPool = pool
	server.EnableTrace = true
	reqCtx := &app.RequestContext{}
	server.Core = &mockCore{
		ctxPool: &sync.Pool{
			New: func() any {
				ti := traceinfo.NewTraceInfo()
				ti.Stats().SetLevel(stats.LevelDetailed)
				reqCtx.SetTraceInfo(&mockTraceInfo{ti})
				return reqCtx
			},
		},
		controller: &internalStats.Controller{},
	}
	err := server.Serve(context.TODO(), mock.NewConn("GET /aaa HTTP/1.1\nHost: foobar.com\n\n"))
	assert.True(t, errors.Is(err, errs.ErrShortConnection))
	traceInfo := reqCtx.GetTraceInfo()
	assert.False(t, traceInfo.Stats().GetEvent(stats.HTTPStart).IsNil())
	assert.False(t, traceInfo.Stats().GetEvent(stats.ReadHeaderStart).IsNil())
	assert.False(t, traceInfo.Stats().GetEvent(stats.ReadHeaderFinish).IsNil())
	assert.False(t, traceInfo.Stats().GetEvent(stats.ReadBodyStart).IsNil())
	assert.False(t, traceInfo.Stats().GetEvent(stats.ReadBodyFinish).IsNil())
	assert.False(t, traceInfo.Stats().GetEvent(stats.ServerHandleStart).IsNil())
	assert.False(t, traceInfo.Stats().GetEvent(stats.ServerHandleFinish).IsNil())
	assert.False(t, traceInfo.Stats().GetEvent(stats.WriteStart).IsNil())
	assert.False(t, traceInfo.Stats().GetEvent(stats.WriteFinish).IsNil())
	assert.False(t, traceInfo.Stats().GetEvent(stats.HTTPFinish).IsNil())
}
