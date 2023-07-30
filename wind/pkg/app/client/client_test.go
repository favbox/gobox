package client

import (
	"testing"

	"github.com/favbox/gosky/wind/pkg/common/config"
	"github.com/favbox/gosky/wind/pkg/route"
)

func TestCloseIdleConnections(t *testing.T) {
	opt := config.NewOptions([]config.Option{})
	opt.Addr = "unix-test-10000"
	opt.Network = "unix"
	engine := route.NewEngine(opt)

}
