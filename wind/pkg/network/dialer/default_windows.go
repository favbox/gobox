package dialer

import "github.com/favbox/gosky/wind/pkg/network/standard"

func init() {
	defaultDialer = standard.NewDialer()
}
