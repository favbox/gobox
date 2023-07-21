package ext

import (
	"bytes"
	"fmt"

	"github.com/favbox/gosky/wind/pkg/network"
)

func MustPeekBuffered(r network.Reader) []byte {
	l := r.Len()
	buf, err := r.Peek(l)
	if len(buf) == 0 || err != nil {
		panic(fmt.Sprintf("bufio.Reader.Peek() 返回异常数据 (%q, %v)", buf, err))
	}

	return buf
}

func MustDiscard(r network.Reader, n int) {
	if err := r.Skip(n); err != nil {
		panic(fmt.Sprintf("bufio.Reader.Discard(%d) failed: %s", n, err))
	}
}

// BufferSnippet 返回字节切片的片段。
//
// 形如: <前缀 20 位>...<后缀=总长度-20位>
//
// 若前缀长 >= 后缀长，则直接返回原始切片。
func BufferSnippet(b []byte) string {
	n := len(b)
	start := 20
	end := n - start
	if start >= end {
		start = n
		end = n
	}
	bStart, bEnd := b[:start], b[end:]
	if len(bEnd) == 0 {
		return fmt.Sprintf("%q", b)
	}
	return fmt.Sprintf("%q...%q", bStart, bEnd)
}

// ReadRawHeaders 读取原始标头。
func ReadRawHeaders(dst, buf []byte) ([]byte, int, error) {
	n := bytes.IndexByte(buf, '\n')
	if n < 0 {
		return dst[:0], 0, errNeedMore
	}
	if (n == 1 && buf[0] == '\r') || n == 0 {
		// 空标头
		return dst, n + 1, nil
	}

	n++
	b := buf
	m := n
	for {
		b = b[m:]
		m = bytes.IndexByte(b, '\n')
		if m < 0 {
			return dst, 0, errNeedMore
		}
		m++
		n += m
		if (m == 2 && b[0] == '\r') || m == 1 {
			dst = append(dst, buf[:n]...)
			return dst, n, nil
		}
	}
}

func normalizeHeaderValue(ov, ob []byte, headerLength int) (nv, nb []byte, nhl int) {
	nv = ov
	length := len(ov)
	if length <= 0 {
		return
	}
	write := 0
	shrunk := 0
	lineStart := false
	for read := 0; read < length; read++ {
		c := ov[read]
		if c == '\r' || c == '\n' {
			shrunk++
			if c == '\n' {
				lineStart = true
			}
			continue
		} else if lineStart && c == '\t' {
			c = ' '
		} else {
			lineStart = false
		}
		nv[write] = c
		write++
	}

	nv = nv[:write]
	copy(ob[write:], ob[write+shrunk:])

	// 检查我们需要跳过 \r\n 还是 \n
	skip := 0
	if ob[write] == '\r' {
		if ob[write+1] == '\n' {
			skip += 2
		} else {
			skip++
		}
	} else if ob[write] == '\n' {
		skip++
	}

	nb = ob[write+skip : len(ob)-shrunk]
	nhl = headerLength - shrunk
	return
}

func stripSpace(b []byte) []byte {
	for len(b) > 0 && b[0] == ' ' {
		b = b[1:]
	}
	for len(b) > 0 && b[len(b)-1] == ' ' {
		b = b[:len(b)-1]
	}
	return b
}

func isOnlyCRLF(b []byte) bool {
	for _, ch := range b {
		if ch != '\r' && ch != '\n' {
			return false
		}
	}
	return true
}
