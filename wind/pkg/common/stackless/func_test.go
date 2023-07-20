package stackless

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewFuncSimple(t *testing.T) {
	t.Parallel()

	var n uint64
	f := NewFunc(func(ctx any) {
		atomic.AddUint64(&n, uint64(ctx.(int)))
	})

	iterations := 4 * 1024
	for i := 0; i < iterations; i++ {
		if !f(2) {
			t.Fatalf("f 不可返回假")
		}
	}
	if n != uint64(2*iterations) {
		t.Fatalf("不期望的 n：%d。期望 %d", n, 2*iterations)
	}
}

func TestNewFuncMulti(t *testing.T) {
	t.Parallel()

	var n1, n2 uint64
	f1 := NewFunc(func(ctx any) {
		atomic.AddUint64(&n1, uint64(ctx.(int)))
	})
	f2 := NewFunc(func(ctx any) {
		atomic.AddUint64(&n2, uint64(ctx.(int)))
	})

	iterations := 4 * 1024

	f1Done := make(chan error, 1)
	go func() {
		var err error
		for i := 0; i < iterations; i++ {
			if !f1(3) {
				err = fmt.Errorf("f1 mustn't return false")
				break
			}
		}
		f1Done <- err
	}()

	f2Done := make(chan error, 1)
	go func() {
		var err error
		for i := 0; i < iterations; i++ {
			if !f2(5) {
				err = fmt.Errorf("f2 mustn't return false")
				break
			}
		}
		f2Done <- err
	}()

	select {
	case err := <-f1Done:
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
	case <-time.After(time.Second):
		t.Fatalf("timeout")
	}

	select {
	case err := <-f2Done:
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
	case <-time.After(time.Second):
		t.Fatalf("timeout")
	}

	if n1 != uint64(3*iterations) {
		t.Fatalf("unexpected n1: %d. Expecting %d", n1, 3*iterations)
	}
	if n2 != uint64(5*iterations) {
		t.Fatalf("unexpected n2: %d. Expecting %d", n2, 5*iterations)
	}
}

func BenchmarkFuncPure(b *testing.B) {
	b.ReportAllocs()

	var n uint64
	f := func(x *uint64) {
		atomic.AddUint64(&n, *x)
	}
	b.RunParallel(func(pb *testing.PB) {
		x := uint64(1)
		for pb.Next() {
			f(&x)
		}
	})
	if n != uint64(b.N) {
		b.Fatalf("unexpected n: %d. Expecting %d", n, b.N)
	}
}

func BenchmarkFuncOverhead(b *testing.B) {
	b.ReportAllocs()

	var n uint64
	f := NewFunc(func(ctx any) {
		atomic.AddUint64(&n, *(ctx.(*uint64)))
	})
	b.RunParallel(func(pb *testing.PB) {
		x := uint64(1)
		for pb.Next() {
			if !f(&x) {
				b.Fatalf("f mustn't return false")
			}
		}
	})
	if n != uint64(b.N) {
		b.Fatalf("unexpected n: %d. Expecting %d", n, b.N)
	}
}
