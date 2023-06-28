package syncx

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func ExampleStreamPool() {
	p := NewStreamPool()

	times := []int{20, 30, 10}
	for _, millis := range times {
		dur := time.Duration(millis) * time.Millisecond
		p.Go(func() StreamCallback {
			time.Sleep(dur)
			return func() {
				fmt.Println(dur)
			}
		})
	}
	p.Wait()

	// Output:
	// 20ms
	// 30ms
	// 10ms
}

func TestStreamPool(t *testing.T) {
	t.Parallel()

	t.Run("简单的有序并发示例", func(t *testing.T) {
		t.Parallel()

		p := NewStreamPool()
		var res []int
		for i := 0; i < 5; i++ {
			i := i
			p.Go(func() StreamCallback {
				i *= 2
				return func() {
					res = append(res, i)
				}
			})
		}
		p.Wait()
		require.Equal(t, []int{0, 2, 4, 6, 8}, res)
	})

	t.Run("限制最大并发数", func(t *testing.T) {
		t.Parallel()

		p := NewStreamPool().WithMaxGoroutines(5)
		var currentTaskCount atomic.Int64
		var currentCallbackCount atomic.Int64
		for i := 0; i < 50; i++ {
			p.Go(func() StreamCallback {
				curr := currentTaskCount.Add(1)
				if curr > 5 {
					t.Fatal("并发数超限")
				}
				defer currentTaskCount.Add(-1)

				time.Sleep(time.Millisecond)

				return func() {
					curr := currentCallbackCount.Add(1)
					if curr > 1 {
						t.Fatal("回调数超限")
					}
					defer currentCallbackCount.Add(-1)
				}
			})
		}
		p.Wait()
	})

	t.Run("传播任务恐慌", func(t *testing.T) {
		t.Parallel()
		p := NewStreamPool()
		p.Go(func() StreamCallback {
			panic("任务恐慌")
		})
		require.Panics(t, p.Wait)
	})

	t.Run("传播回调恐慌", func(t *testing.T) {
		t.Parallel()
		s := NewStreamPool()
		s.Go(func() StreamCallback {
			return func() {
				panic("回调恐慌")
			}
		})
		require.Panics(t, s.Wait)
	})

	t.Run("回调恐慌不阻碍任务执行", func(t *testing.T) {
		t.Parallel()
		p := NewStreamPool().WithMaxGoroutines(5)
		p.Go(func() StreamCallback {
			return func() {
				panic("回调恐慌")
			}
		})
		for i := 0; i < 100; i++ {
			p.Go(func() StreamCallback {
				return func() {}
			})
		}
		require.Panics(t, p.Wait)
	})
}

func BenchmarkStreamPool(b *testing.B) {
	b.Run("整体开销", func(b *testing.B) {
		b.ReportAllocs()

		f := func() {}
		for i := 0; i < b.N; i++ {
			p := NewStreamPool()
			p.Go(func() StreamCallback { return f })
			p.Wait()
		}
	})

	b.Run("提交开销", func(b *testing.B) {
		b.ReportAllocs()

		f := func() {}
		p := NewStreamPool()
		for i := 0; i < b.N; i++ {
			p.Go(func() StreamCallback { return f })
		}
		p.Wait()
	})
}
