package network

import "context"

type Transporter interface {
	// ListenAndServe 监听并准备接收连接。
	ListenAndServe(OnData) error

	// Close 立即关闭传输器。
	Close() error

	// Shutdown 平滑关闭传输器。
	Shutdown(ctx context.Context) error
}

// OnData 连接数据准备完毕时的回调函数。
type OnData func(ctx context.Context, conn any) error
