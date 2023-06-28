package bytebufferpool

import (
	"bytes"
	"fmt"
	"io"
	"testing"
)

func TestByteBuffer_ReadFrom(t *testing.T) {
	prefix := "foobar"
	expectedS := "asdfasdf"
	prefixLen := int64(len(prefix))
	expectedN := int64(len(expectedS))

	var bb ByteBuffer
	bb.WriteString(prefix)

	rf := (io.ReaderFrom)(&bb)
	for i := 0; i < 20; i++ {
		r := bytes.NewBufferString(expectedS)
		n, err := rf.ReadFrom(r)
		if n != expectedN {
			t.Fatalf("异常值 n=%d。 期望 %d。 迭代 %d", n, expectedN, i)
		}
		if err != nil {
			t.Fatalf("异常：%s", err)
		}
		bbLen := int64(bb.Len())
		expectedLen := prefixLen + int64(i+1)*expectedN
		if bbLen != expectedLen {
			t.Fatalf("字节缓冲器长度异常：%d。期望：%d", bbLen, expectedLen)
		}
		for j := 0; j < i; j++ {
			start := prefixLen + int64(j)*expectedN
			b := bb.B[start : start+expectedN]
			if string(b) != expectedS {
				t.Fatalf("异常的缓冲区内容：%q。期待 %q", b, expectedS)
			}
		}
	}
}

func TestByteBuffer_WriteTo(t *testing.T) {
	expectedS := "foobarbaz"
	var bb ByteBuffer
	bb.WriteString(expectedS[:3])
	bb.WriteString(expectedS[3:])

	wt := (io.WriterTo)(&bb)
	var w bytes.Buffer
	for i := 0; i < 10; i++ {
		n, err := wt.WriteTo(&w)
		if n != int64(len(expectedS)) {
			t.Fatalf("WriteTo 返回的n值异常：%d。期望：%d", n, len(expectedS))
		}
		if err != nil {
			t.Fatalf("异常错误：%s", err)
		}
		s := w.String()
		if s != expectedS {
			t.Fatalf("异常字符串已写入 %q。期待 %q", s, expectedS)
		}
		w.Reset()
	}
}

func TestName(t *testing.T) {
	b := ByteBuffer{}

	b.B = []byte("hi")
	fmt.Printf("%p\n", b.B)
	b.B = []byte("hi")
	fmt.Printf("%p\n", b.B)

	b.Set([]byte("hi3"))
	fmt.Printf("%p\n", b.B)
	b.Set([]byte("hi4"))
	fmt.Printf("%p\n", b.B)
	b.Set([]byte("hi5"))
	fmt.Printf("%p\n", b.B)
}
