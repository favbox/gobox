package route

import (
	"net/http"
	"testing"

	"github.com/favbox/gobox/wind/pkg/common/test/assert"
)

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
