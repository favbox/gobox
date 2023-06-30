package errors

import "errors"

var (
	ErrConnectionClosed = errors.New("连接已关闭")
)
