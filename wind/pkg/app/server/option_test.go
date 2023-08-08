package server

import (
	"testing"
	"time"

	"github.com/favbox/gosky/wind/pkg/app/server/registry"
	"github.com/favbox/gosky/wind/pkg/common/config"
	"github.com/favbox/gosky/wind/pkg/common/test/assert"
	"github.com/favbox/gosky/wind/pkg/common/tracer/stats"
	"github.com/favbox/gosky/wind/pkg/common/utils"
)

func TestOptions(t *testing.T) {
	info := &registry.Info{
		ServiceName: "wind.test.api",
		Addr:        utils.NewNetAddr("", ""),
		Weight:      10,
	}
	opt := config.NewOptions([]config.Option{
		WithReadTimeout(time.Second),
		WithWriteTimeout(time.Second),
		WithIdleTimeout(time.Second),
		WithKeepAliveTimeout(time.Second),
		WithRedirectTrailingSlash(false),
		WithRedirectFixedPath(true),
		WithHandleMethodNotAllowed(true),
		WithUseRawPath(true),
		WithRemoveExtraSlash(true),
		WithUnescapePathValues(false),
		WithDisablePreParseMultipartForm(true),
		WithStreamBody(false),
		WithHostPorts(":8888"),
		WithBasePath("/"),
		WithMaxRequestBodySize(2),
		WithDisablePrintRoute(true),
		WithNetwork("unix"),
		WithExitWaitTime(time.Second),
		WithMaxKeepBodySize(500),
		WithGetOnly(true),
		WithKeepAlive(false),
		WithTLS(nil),
		WithH2C(true),
		WithReadBufferSize(100),
		WithALPN(true),
		WithTraceLevel(stats.LevelDisabled),
		WithRegistry(nil, info),
		WithAutoReloadRender(true, 5*time.Second),
	})

	assert.DeepEqual(t, opt.ReadTimeout, time.Second)
	assert.DeepEqual(t, opt.WriteTimeout, time.Second)
	assert.DeepEqual(t, opt.IdleTimeout, time.Second)
	assert.DeepEqual(t, opt.KeepAliveTimeout, time.Second)
	assert.DeepEqual(t, opt.RedirectTrailingSlash, false)
	assert.DeepEqual(t, opt.RedirectFixedPath, true)
	assert.DeepEqual(t, opt.HandleMethodNotAllowed, true)
	assert.DeepEqual(t, opt.UseRawPath, true)
	assert.DeepEqual(t, opt.RemoveExtraSlash, true)
	assert.DeepEqual(t, opt.UnescapePathValues, false)
	assert.DeepEqual(t, opt.DisablePreParseMultipartForm, true)
	assert.DeepEqual(t, opt.StreamRequestBody, false)
	assert.DeepEqual(t, opt.Addr, ":8888")
	assert.DeepEqual(t, opt.BasePath, "/")
	assert.DeepEqual(t, opt.MaxRequestBodySize, 2)
	assert.DeepEqual(t, opt.DisablePrintRoute, true)
	assert.DeepEqual(t, opt.Network, "unix")
	assert.DeepEqual(t, opt.ExitWaitTimeout, time.Second)
	assert.DeepEqual(t, opt.MaxKeepBodySize, 500)
	assert.DeepEqual(t, opt.GetOnly, true)
	assert.DeepEqual(t, opt.DisableKeepalive, true)
	assert.DeepEqual(t, opt.H2C, true)
	assert.DeepEqual(t, opt.ReadBufferSize, 100)
	assert.DeepEqual(t, opt.ALPN, true)
	assert.DeepEqual(t, opt.TraceLevel, stats.LevelDisabled)
	assert.DeepEqual(t, opt.RegistryInfo, info)
	assert.DeepEqual(t, opt.Registry, nil)
	assert.DeepEqual(t, opt.AutoReloadRender, true)
	assert.DeepEqual(t, opt.AutoReloadInterval, 5*time.Second)
}

func TestDefaultOptions(t *testing.T) {
	opt := config.NewOptions([]config.Option{})
	assert.DeepEqual(t, opt.ReadTimeout, time.Minute*3)
	assert.DeepEqual(t, opt.IdleTimeout, time.Minute*3)
	assert.DeepEqual(t, opt.KeepAliveTimeout, time.Minute)
	assert.DeepEqual(t, opt.RedirectTrailingSlash, true)
	assert.DeepEqual(t, opt.RedirectFixedPath, false)
	assert.DeepEqual(t, opt.HandleMethodNotAllowed, false)
	assert.DeepEqual(t, opt.UseRawPath, false)
	assert.DeepEqual(t, opt.RemoveExtraSlash, false)
	assert.DeepEqual(t, opt.UnescapePathValues, true)
	assert.DeepEqual(t, opt.DisablePreParseMultipartForm, false)
	assert.DeepEqual(t, opt.StreamRequestBody, false)
	assert.DeepEqual(t, opt.Addr, ":8888")
	assert.DeepEqual(t, opt.BasePath, "/")
	assert.DeepEqual(t, opt.MaxRequestBodySize, 4*1024*1024)
	assert.DeepEqual(t, opt.GetOnly, false)
	assert.DeepEqual(t, opt.DisableKeepalive, false)
	assert.DeepEqual(t, opt.DisablePrintRoute, false)
	assert.DeepEqual(t, opt.Network, "tcp")
	assert.DeepEqual(t, opt.ExitWaitTimeout, time.Second*5)
	assert.DeepEqual(t, opt.MaxKeepBodySize, 4*1024*1024)
	assert.DeepEqual(t, opt.H2C, false)
	assert.DeepEqual(t, opt.ReadBufferSize, 4096)
	assert.DeepEqual(t, opt.ALPN, false)
	assert.DeepEqual(t, opt.Registry, registry.NoopRegistry)
	assert.DeepEqual(t, opt.AutoReloadRender, false)
	assert.Assert(t, opt.RegistryInfo == nil)
	assert.DeepEqual(t, opt.AutoReloadRender, false)
	assert.DeepEqual(t, opt.AutoReloadInterval, time.Duration(0))
}
