package route

import "github.com/favbox/gosky/wind/pkg/network/standard"

func init() {
	defaultTransporter = standard.NewTransporter
}
