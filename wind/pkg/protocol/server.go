package protocol

import (
	"context"

	"github.com/favbox/gosky/wind/pkg/network"
)

// Server 定义协议层普通连接服务器，只需实现 Serve 方法即可。
type Server interface {
	Serve(ctx context.Context, conn network.Conn) error
}

// StreamServer 定义协议层流式连接服务器，只需实现 Serve 方法接口。
type StreamServer interface {
	Serve(ctx context.Context, conn network.StreamConn) error
}
