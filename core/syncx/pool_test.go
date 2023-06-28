package syncx

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/favbox/gobox/http/pkg/common/test/assert"
	"github.com/zeromicro/go-zero/core/mr"
)

func TestMr(t *testing.T) {
	t.Parallel()

	t.Run("默认不限协程数量，有多少个任务就派生多少个协程", func(t *testing.T) {
		t.Parallel()

		var count atomic.Int64
		var tasks []func()
		for i := 0; i < 100; i++ {
			tasks = append(tasks, func() {
				time.Sleep(time.Millisecond)
				count.Add(1)
			})
		}
		mr.FinishVoid(tasks...)
		assert.True(t, count.Load() == int64(100))
	})

	t.Run("判断协程结果", func(t *testing.T) {
		var count atomic.Int64

		mr.FinishVoid(
			func() { count.Add(1) },
			func() { count.Add(2) },
		)

		assert.True(t, count.Load() == int64(3))
	})
}

func TestPool(t *testing.T) {
	t.Parallel()

	t.Run("默认不限协程数量，有多少任务就有多少协程", func(t *testing.T) {
		t.Parallel()

		var p Pool
		var count atomic.Int64
		for i := 0; i < 100; i++ {
			p.Go(func() {
				time.Sleep(time.Millisecond)
				count.Add(1)
			})
		}
		p.Wait()
		assert.True(t, count.Load() == int64(100))
	})

	t.Run("判断协程结果是否准确", func(t *testing.T) {
		t.Parallel()

		var p Pool
		var count atomic.Int64

		p.Go(func() { count.Add(1) })
		p.Go(func() { count.Add(2) })
		p.Wait()

		assert.True(t, count.Load() == int64(3))
	})

	t.Run("协程的错误会被传播", func(t *testing.T) {
		t.Parallel()

		p := NewPool()
		for i := 0; i < 10; i++ {
			i := i
			p.Go(func() {
				if i == 5 {
					panic(i)
				}
			})
		}
		assert.Panic(t, p.Wait)
	})

	t.Run("获取的协程数量要与设置的一致", func(t *testing.T) {
		t.Parallel()
		g := NewPool().WithMaxGoroutines(365)
		assert.True(t, g.MaxGoroutines() == 365)
	})

	t.Run("限制协程数量", func(t *testing.T) {
		t.Parallel()

		for _, maxConcurrent := range []int{1, 10, 100} {
			p := NewPool().WithMaxGoroutines(maxConcurrent)

			var count atomic.Int64

			taskCount := maxConcurrent * 10
			for i := 0; i < taskCount; i++ {
				p.Go(func() {
					cur := count.Add(1)
					if cur > int64(maxConcurrent) {
						panic("超过最大并发数")
					}
					time.Sleep(time.Millisecond)
					count.Add(-1)
				})
			}
			p.Wait()
		}
	})
}

func BenchmarkPool(b *testing.B) {
	f := func() {}

	b.Run("mr.Finish 单任务提交开销评估【单核】", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			mr.FinishVoid(f)
		}
	})

	b.Run("syncx.Pool 单任务提交开销评估【单核】", func(b *testing.B) {
		b.ReportAllocs()
		p := NewPool()
		for i := 0; i < b.N; i++ {
			p.Go(f)
		}
		p.Wait()
	})

	b.Run("syncx.Pool 单任务整体开销评估【单核】", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			p := NewPool()
			p.Go(f)
			p.Wait()
		}
	})

	b.Run("mr.Finish 单任务提交开销评估【多核】", func(b *testing.B) {
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				mr.FinishVoid(f)
			}
		})
	})

	b.Run("syncx.Pool 单任务提交开销评估【多核】", func(b *testing.B) {
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			p := NewPool()
			for pb.Next() {
				p.Go(f)
			}
			p.Wait()
		})
	})
}
