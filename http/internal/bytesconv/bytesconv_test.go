package bytesconv

import (
	"testing"

	"github.com/favbox/gobox/http/pkg/common/test/assert"
)

func TestLowercaseBytes(t *testing.T) {
	t.Parallel()

	for _, v := range []struct {
		b1, b2 []byte
	}{
		{[]byte("GOBOX-HTTP"), []byte("gobox-http")},
		{[]byte("GOBOX"), []byte("gobox")},
		{[]byte("HTTP"), []byte("http")},
	} {
		LowercaseBytes(v.b1)
		assert.DeepEqual(t, v.b2, v.b1)
	}
}

func TestB2s(t *testing.T) {
	t.Parallel()

	for _, v := range []struct {
		s string
		b []byte
	}{
		{"gobox-http", []byte("gobox-http")},
		{"gobox", []byte("gobox")},
		{"http", []byte("http")},
	} {
		assert.DeepEqual(t, v.s, B2s(v.b))
	}
}

func TestS2b(t *testing.T) {
	t.Parallel()

	for _, v := range []struct {
		s string
		b []byte
	}{
		{"gobox-http", []byte("gobox-http")},
		{"gobox", []byte("gobox")},
		{"http", []byte("http")},
	} {
		assert.DeepEqual(t, v.b, S2b(v.s))
	}
}

func BenchmarkB2s(b *testing.B) {
	s := "hi"
	bs := []byte("hi")

	b.Run("std/string", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = string(bs)
		}
	})

	b.Run("bytesconv/B2s", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = B2s(bs)
		}
	})

	b.Run("std/[]byte", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = []byte(s)
		}
	})

	b.Run("bytesconv/S2b", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = S2b(s)
		}
	})

	b.Run("bytesconv/B2s", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = B2s(bs)
		}
	})

	b.Run("std/string multicore", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_ = string(bs)
			}
		})
	})

	b.Run("bytesconv/B2s multicore", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_ = B2s(bs)
			}
		})
	})
}
