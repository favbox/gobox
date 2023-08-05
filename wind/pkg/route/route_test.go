package route

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/favbox/gosky/wind/pkg/app"
	"github.com/favbox/gosky/wind/pkg/common/test/assert"
	"github.com/favbox/gosky/wind/pkg/protocol"
)

type header struct {
	Key   string
	Value string
}

func doRequest(e *Engine, method, path string, headers ...header) *httptest.ResponseRecorder {
	// 上下文
	ctx := e.ctxPool.Get().(*app.RequestContext)
	ctx.HTMLRender = e.htmlRender

	// 请求
	req := protocol.NewRequest(method, path, nil)
	req.CopyTo(&ctx.Request)
	for _, v := range headers {
		ctx.Request.Header.Add(v.Key, v.Value)
	}

	// 服务
	e.ServeHTTP(context.Background(), ctx)

	w := httptest.NewRecorder()
	h := w.Header()
	ctx.Response.Header.VisitAll(func(key, value []byte) {
		h.Add(string(key), string(value))
	})
	w.WriteHeader(ctx.Response.StatusCode())
	if _, err := w.Write(ctx.Response.Body()); err != nil {
		panic(err)
	}
	ctx.Reset()
	e.ctxPool.Put(ctx)

	return w
}

func TestRouterGroup_BadMethod(t *testing.T) {
	r := &RouterGroup{
		Handlers: nil,
		basePath: "/",
		root:     true,
	}
	assert.Panic(t, func() { r.Handle(http.MethodGet, "/") })
	assert.Panic(t, func() { r.Handle(" GET", "/") })
	assert.Panic(t, func() { r.Handle("GET ", "/") })
	assert.Panic(t, func() { r.Handle("", "/") })
	assert.Panic(t, func() { r.Handle("PO ST", "/") })
	assert.Panic(t, func() { r.Handle("1GET", "/") })
	assert.Panic(t, func() { r.Handle("PATch", "/") })
}
