package syncx

import (
	"runtime"
	"sync/atomic"
)

// Iterator 零值可用、并发安全、协程数量可控的迭代器。
//
// 建议用于耗时任务，不建议用于短时任务。
type Iterator[T any] struct {
	// 最大协程数，默认为 runtime.GOMAXPROCS(0)
	MaxGoroutines int
}

// ForEach 以回调的方式遍历输入的切片值。
//
// 使用默认协程数，如需自定义可使用 Iterator.ForEach。
func ForEach[T any](inputs []T, f func(*T)) {
	Iterator[T]{}.ForEach(inputs, f)
}

// ForEach 以回调的方式遍历输入的切片值。
func (iter Iterator[T]) ForEach(inputs []T, f func(*T)) {
	iter.ForEachIdx(inputs, func(_ int, t *T) { f(t) })
}

// ForEachIdx 以回调的方式遍历输入的索引及切片元素。
//
// 使用默认协程数，如需自定义可使用 Iterator.ForEachIdx。
func ForEachIdx[T any](inputs []T, f func(int, *T)) {
	Iterator[T]{}.ForEachIdx(inputs, f)
}

// ForEachIdx 以回调的方式遍历输入的索引及切片值。
func (iter Iterator[T]) ForEachIdx(inputs []T, f func(int, *T)) {
	// 设置默认的最大协程数
	if iter.MaxGoroutines == 0 {
		// iter 是值接收器，可安全改变
		iter.MaxGoroutines = defaultMaxGoroutines()
	}

	// 限制最大协程数量不能超过输入元素的数量
	inputCount := len(inputs)
	if iter.MaxGoroutines > inputCount {
		iter.MaxGoroutines = inputCount
	}

	// 循坏外构建任务，以免额外的闭包分配
	var idx atomic.Int64
	task := func() {
		i := int(idx.Add(1) - 1)
		for ; i < inputCount; i = int(idx.Add(1) - 1) {
			f(i, &inputs[i])
		}
	}

	// 循坏提交并发任务
	var wg WaitGroup
	for i := 0; i < iter.MaxGoroutines; i++ {
		wg.Go(task)
	}
	wg.Wait()
}

// 迭代器的默认最大协程数，即当前进程可用的逻辑 CPU 数量。
func defaultMaxGoroutines() int {
	return runtime.GOMAXPROCS(0)
}
