package utils

import "sync"

// CopyBufPool 拷贝缓冲池。
var CopyBufPool = sync.Pool{
	New: func() any {
		return make([]byte, 4096)
	},
}
