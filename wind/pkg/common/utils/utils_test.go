package utils

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	lowerStr = []byte("content-length")
	upperStr = []byte("Content-Length")
)

func TestCaseInsensitiveCompare(t *testing.T) {
	assert.True(t, CaseInsensitiveCompare(lowerStr, upperStr))
	assert.False(t, CaseInsensitiveCompare(lowerStr, []byte("content-type")))
}

func caseInsensitiveCompare(a, b []byte) bool {
	return bytes.Equal(bytes.ToLower(a), bytes.ToLower(b))
}

func BenchmarkCaseInsensitiveCompare(b *testing.B) {
	for i := 0; i < b.N; i++ {
		CaseInsensitiveCompare(lowerStr, upperStr)
	}
}

func Benchmark_caseInsensitiveCompare(b *testing.B) {
	for i := 0; i < b.N; i++ {
		caseInsensitiveCompare(lowerStr, upperStr)
	}
}

func TestName(t *testing.T) {
	a := []byte("hi")
	fmt.Printf("%s %p\n", a, &a)

	a = []byte("hello")
	fmt.Printf("%s %p\n", a, &a)

	a = append(a[:0], "hey"...)
	fmt.Printf("%s %p\n", a, &a)
}

var a []byte

func BenchmarkAssign(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		a = []byte("hi")
	}
}

func BenchmarkAppendAssign(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		a = append(a[:0], "hey"...)
	}
}
