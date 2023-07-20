package utils

import (
	"io"

	"github.com/favbox/gobox/wind/pkg/network"
)

// CopyZeroAlloc 零分配拷贝。
func CopyZeroAlloc(w network.Writer, r io.Reader) (int64, error) {
	vBuf := CopyBufPool.Get()
	buf := vBuf.([]byte)
	n, err := CopyBuffer(w, r, buf)
	CopyBufPool.Put(vBuf)
	return n, err
}

func CopyBuffer(dst network.Writer, src io.Reader, buf []byte) (written int64, err error) {
	if buf != nil && len(buf) == 0 {
		panic("CopyBuffer 中的 buf 缓冲区为空")
	}
	return copyBuffer(dst, src, buf)
}

// copyBuffer 是 CopyZeroAlloc 和 CopyBuffer 的实际实现。
// 若 buf 为空，会分配一个。
func copyBuffer(dst network.Writer, src io.Reader, buf []byte) (written int64, err error) {
	if wt, ok := src.(io.WriterTo); ok {
		if w, ok := dst.(io.Writer); ok {
			return wt.WriteTo(w)
		}
	}

	// Sendfile 实现
	if rf, ok := dst.(io.ReaderFrom); ok {
		return rf.ReadFrom(src)
	}

	if buf == nil {
		size := 32 * 1024
		if l, ok := src.(*io.LimitedReader); ok && int64(size) > l.N {
			if l.N < 1 {
				size = 1
			} else {
				size = int(l.N)
			}
		}
		buf = make([]byte, size)
	}
	for {
		nr, er := src.Read(buf)
		if nr > 0 {
			nw, eb := dst.WriteBinary(buf[:nr])
			if eb != nil {
				err = eb
				return
			}

			if nw > 0 {
				written += int64(nw)
			}
			if nr != nw {
				err = io.ErrShortWrite
				return
			}
			if err = dst.Flush(); err != nil {
				return
			}
		}
		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
	}

	return
}
