package bytebufferpool

import (
	"io"
)

// ByteBuffer 提供字节缓冲区，可最小化内存分配。
//
// ByteBuffer 可将数据附加到字节切片的函数。
type ByteBuffer struct {
	// B 是用于 append 操作的缓冲区。
	B []byte
}

// Len 返回字节缓冲区的大小。
func (b *ByteBuffer) Len() int {
	return len(b.B)
}

// ReadFrom 实现 io.ReaderFrom。
//
// 将从 r 读取的所有数据附加到 b。
func (b *ByteBuffer) ReadFrom(r io.Reader) (int64, error) {
	p := b.B
	nStart := int64(len(p))
	nMax := int64(cap(p))
	n := nStart
	if nMax == 0 {
		nMax = 64
		p = make([]byte, nMax)
	} else {
		p = p[:nMax]
	}
	for {
		if n == nMax {
			nMax *= 2
			bNew := make([]byte, nMax)
			copy(bNew, p)
			p = bNew
		}
		nn, err := r.Read(p[n:])
		n += int64(nn)
		if err != nil {
			b.B = p[:n]
			n -= nStart
			if err == io.EOF {
				return n, nil
			}
			return n, err
		}
	}
}

// WriteTo 实现 io.WriterTo。
//
// 将 b 中所有数据写入 w。
func (b *ByteBuffer) WriteTo(w io.Writer) (int64, error) {
	n, err := w.Write(b.B)
	return int64(n), err
}

// Bytes 返回 b.B，即缓冲区累计的所有字节。
func (b *ByteBuffer) Bytes() []byte {
	return b.B
}

// Bytes 实现 io.Writer - 附加字节切片 p 到缓冲区。
//
// 返回附加字节切片的长度和一个 nil error。
func (b *ByteBuffer) Write(p []byte) (int, error) {
	b.B = append(b.B, p...)
	return len(p), nil
}

// WriteByte 附加字节 c 到缓冲区。
//
// 函数的目的是为了兼容 bytes.Buffer。
//
// 函数总是返回 nil error。
func (b *ByteBuffer) WriteByte(c byte) error {
	b.B = append(b.B, c)
	return nil
}

// WriteString 附加字符串 s 到缓冲区。
//
// 返回附加字符串的长度和 nil error。
func (b *ByteBuffer) WriteString(s string) (int, error) {
	b.B = append(b.B, s...)
	return len(s), nil
}

// Set 将缓冲区设置为字节切片 p。
//
// Set 和 = 赋值不同，不会开辟新的底层存储。
func (b *ByteBuffer) Set(p []byte) {
	b.B = append(b.B[:0], p...)
}

// SetString 将缓冲区设置为字符串 s。
func (b *ByteBuffer) SetString(s string) {
	b.B = append(b.B[:0], s...)
}

// 返回字节缓冲区的字符串表达形式。
func (b *ByteBuffer) String() string {
	//return bytesconv.B2s(b.B)
	return string(b.B)
}

// Reset 将缓冲区重置为空，但保留底层存储以供将来写入使用。
func (b *ByteBuffer) Reset() {
	b.B = b.B[:0]
}
