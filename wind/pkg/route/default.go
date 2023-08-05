//go:build !windows

package route

import "github.com/favbox/gosky/wind/pkg/network/netpoll"

func init() {
	defaultTransporter = netpoll.NewTransporter
}
