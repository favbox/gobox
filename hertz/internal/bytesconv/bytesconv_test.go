package bytesconv

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/favbox/gobox/hertz/pkg/common/test/assert"
)

func TestAppendQuotedArg(t *testing.T) {
	t.Parallel()

	// 与 url.QueryEscape 同步
	allCases := make([]byte, 256)
	for i := 0; i < 256; i++ {
		allCases[i] = byte(i)
	}
	res := B2s(AppendQuotedArg(nil, allCases))
	expect := url.QueryEscape(B2s(allCases))
	assert.DeepEqual(t, expect, res)
}

func TestAppendQuotedPath(t *testing.T) {
	t.Parallel()

	// 测试所有字符
	pathSegment := make([]byte, 256)
	for i := 0; i < 256; i++ {
		pathSegment[i] = byte(i)
	}
	for _, s := range []struct {
		path string
	}{
		{"/"},
		{"//"},
		{"/foo/bar"},
		{"*"},
		{"/foo/" + B2s(pathSegment)},
	} {
		u := url.URL{Path: s.path}
		expectedS := u.EscapedPath()
		res := B2s(AppendQuotedPath(nil, S2b(s.path)))
		assert.DeepEqual(t, expectedS, res)
	}
}

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

func TestAppendUint(t *testing.T) {
	t.Parallel()

	for _, s := range []struct {
		n int
	}{
		{0},
		{123},
		{0x7fffffff},
	} {
		expectedS := fmt.Sprintf("%d", s.n)
		s := AppendUint(nil, s.n)
		assert.DeepEqual(t, expectedS, B2s(s))
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

func BenchmarkAppendQuotedArg(b *testing.B) {
	allCases := make([]byte, 256)
	for i := 0; i < 256; i++ {
		allCases[i] = byte(i)
	}

	b.Run("AppendQuotedArg", func(b *testing.B) {
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_ = B2s(AppendQuotedArg(nil, allCases))
			}
		})
	})

	b.Run("url.QueryEscape", func(b *testing.B) {
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_ = url.QueryEscape(B2s(allCases))
			}
		})
	})
}

func BenchmarkAppendQuotedPath(b *testing.B) {
	allCases := make([]byte, 256)
	for i := 0; i < 256; i++ {
		allCases[i] = byte(i)
	}
	u := url.URL{Path: B2s(allCases)}

	b.Run("AppendQuotedPath", func(b *testing.B) {
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_ = B2s(AppendQuotedPath(nil, allCases))
			}
		})
	})

	b.Run("url.QueryEscape", func(b *testing.B) {
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_ = u.EscapedPath()
			}
		})
	})
}
