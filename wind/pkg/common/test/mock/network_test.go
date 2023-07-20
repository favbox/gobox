package mock

import (
	"context"
	"testing"
	"time"

	"github.com/cloudwego/netpoll"
	errs "github.com/favbox/gobox/wind/pkg/common/errors"
	"github.com/favbox/gobox/wind/pkg/common/test/assert"
)

func TestConn(t *testing.T) {
	t.Run("TestReader", func(t *testing.T) {
		s1 := "abcdef4343"
		conn1 := NewConn(s1)
		err := conn1.SetReadDeadline(time.Now().Add(time.Millisecond * 100))
		assert.DeepEqual(t, nil, err)
		err = conn1.SetReadTimeout(time.Millisecond * 100)
		assert.DeepEqual(t, nil, err)

		// Peek Skip Read
		b, _ := conn1.Peek(1)
		assert.DeepEqual(t, []byte{'a'}, b)
		conn1.Skip(1)                   // 游标跳过了 a
		readByte, _ := conn1.ReadByte() // 游标跳过了 b
		assert.DeepEqual(t, byte('b'), readByte)

		p := make([]byte, 100)
		n, err := conn1.Read(p) // 从 c 开始读取 100 个字节
		assert.DeepEqual(t, nil, err)
		assert.DeepEqual(t, s1[2:], string(p[:n]))

		_, err = conn1.Peek(1) // 上一步已经读到底了，此步骤取不出来
		assert.DeepEqual(t, errs.ErrTimeout, err)

		conn2 := NewConn(s1)             // 重新来
		p, _ = conn2.ReadBinary(len(s1)) // 一次性读完
		assert.DeepEqual(t, s1, string(p))
		assert.DeepEqual(t, 0, conn2.Len()) // 没有可读的了
		// Reader
		assert.DeepEqual(t, conn2.zr, conn2.Reader())
	})

	t.Run("TestReadWriter", func(t *testing.T) {
		s1 := "abcdef4343"
		conn := NewConn(s1)
		p, err := conn.ReadBinary(len(s1)) // 一次性全读出来
		assert.DeepEqual(t, nil, err)
		assert.DeepEqual(t, s1, string(p))

		wr := conn.WriterRecorder()
		s2 := "efghljk"
		// WriteBinary
		n, err := conn.WriteBinary([]byte(s2)) // 写入缓冲区
		assert.DeepEqual(t, nil, err)
		assert.DeepEqual(t, len(s2), n)
		assert.DeepEqual(t, len(s2), wr.WroteLen())

		// Flush
		p, _ = wr.ReadBinary(len(s2)) // 此时上一步写入的数据还在缓冲区，所以读不出来
		assert.DeepEqual(t, len(p), 0)

		conn.Flush()                  // 将缓冲区数据发至对端
		p, _ = wr.ReadBinary(len(s2)) // 可以读出来了
		assert.DeepEqual(t, s2, string(p))

		// Write
		s3 := "foobarbaz"
		n, err = conn.Write([]byte(s3)) // 直接写入对端
		assert.DeepEqual(t, nil, err)
		assert.DeepEqual(t, len(s3), n)
		p, _ = wr.ReadBinary(len(s3))
		assert.DeepEqual(t, s3, string(p))

		// Malloc
		buf, _ := conn.Malloc(10)
		assert.DeepEqual(t, 10, len(buf))
		// Writer
		assert.DeepEqual(t, conn.zw, conn.Writer())

		_, err = DialerFun("")
		assert.DeepEqual(t, nil, err)
	})

	t.Run("TestNotImplement", func(t *testing.T) {
		conn := NewConn("")
		t1 := time.Now().Add(time.Millisecond)
		du1 := time.Second
		assert.DeepEqual(t, nil, conn.Release())
		assert.DeepEqual(t, nil, conn.Close())
		assert.DeepEqual(t, nil, conn.LocalAddr())
		assert.DeepEqual(t, nil, conn.RemoteAddr())
		assert.DeepEqual(t, nil, conn.SetIdleTimeout(du1))
		assert.Panic(t, func() {
			conn.SetDeadline(t1)
		})
		assert.Panic(t, func() {
			conn.SetWriteDeadline(t1)
		})
		assert.Panic(t, func() {
			conn.IsActive()
		})
		assert.Panic(t, func() {
			conn.SetOnRequest(func(ctx context.Context, connection netpoll.Connection) error {
				return nil
			})
		})
		assert.Panic(t, func() {
			conn.AddCloseCallback(func(connection netpoll.Connection) error {
				return nil
			})
		})
	})
}
