package syncx

import (
	"fmt"
	"testing"
	"time"

	"github.com/zeromicro/go-zero/core/mr"
)

func TestIter(t *testing.T) {
	inputs := []int{1, 2, 3, 4, 5}
	ForEachIdx(inputs, func(idx int, v *int) {
		fmt.Println(*v)
	})
}

func BenchmarkIter(b *testing.B) {
	var inputs []int
	num := 1000
	for i := 0; i < num; i++ {
		inputs = append(inputs, i)
	}

	task := func(int, *int) {
		time.Sleep(10 * time.Millisecond)
	}

	b.Run("并发遍历", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			Iterator[int]{num}.ForEachIdx(inputs, task)
		}
	})

	b.Run("go-zero 遍历", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			mr.ForEach(func(source chan<- int) {
				for i := 0; i < num; i++ {
					source <- i
				}
			}, func(item int) {
				task(0, &item)
			}, mr.WithWorkers(num))
		}
	})

	b.Run("普通遍历", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			foreach(inputs, task)
		}
	})

	b.Run("并发遍历[多核]", func(b *testing.B) {
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				Iterator[int]{num}.ForEachIdx(inputs, task)
			}
		})
	})

	b.Run("go-zero 并发遍历[多核]", func(b *testing.B) {
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				mr.ForEach(func(source chan<- int) {
					for i := 0; i < num; i++ {
						source <- i
					}
				}, func(item int) {
					task(0, &item)
				}, mr.WithWorkers(num))
			}
		})
	})

	b.Run("普通遍历[多核]", func(b *testing.B) {
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				foreach(inputs, task)
			}
		})
	})
}

func foreach[T any](inputs []T, f func(int, *T)) {
	for i := 0; i < len(inputs); i++ {
		f(i, &inputs[i])
	}
}
