package mock

import (
	"bufio"
	"bytes"
)

// ZeroCopyReader 用于为测试创建的 ZeroCopyReader。
//
// 注意：原则上，单元测试模拟测试时应使用 netpoll.NewReader 创建的 zcReader，
// 但因为其未实现 io.Reader 接口，所以 io.Reader 的测试要求被此结构体取代。
type ZeroCopyReader struct {
	*bufio.Reader
}

func (m ZeroCopyReader) Peek(n int) ([]byte, error) {
	b, err := m.Reader.Peek(n)
	// 若 n 大于 m.Reader 中的缓冲区，
	// 就只会返回 bufio.ErrBufferFull，哪怕底层阅读器返回了 io.EOF。
	// 所以我们用另一个 Peek 来获取真实错误。
	// 了解详情 https://github.com/golang/go/issues/50569
	if err == bufio.ErrBufferFull && len(b) == 0 {
		return m.Reader.Peek(1)
	}
	return b, err
}

func (m ZeroCopyReader) Skip(n int) (err error) {
	_, err = m.Reader.Discard(n)
	return
}

func (m ZeroCopyReader) Release() (err error) {
	return nil
}

func (m ZeroCopyReader) Len() (length int) {
	return m.Reader.Buffered()
}

func (m ZeroCopyReader) ReadBinary(n int) (p []byte, err error) {
	panic("implement me")
}

func NewZeroCopyReader(r string) ZeroCopyReader {
	br := bufio.NewReaderSize(bytes.NewBufferString(r), len(r))
	return ZeroCopyReader{br}
}
