package syncx

import (
	"context"
	"sync"
)

// NewResultPool 创建新的能返回结果的多任务并发协程池。
func NewResultPool[T any]() *ResultPool[T] {
	return &ResultPool[T]{
		pool: *NewPool(),
	}
}

// ResultPool 是能返回泛型结果的多任务并发协程池。
//
// 用 Go() 提交一个任务，用 Wait() 阻塞并等待所有任务完成。
//
// 不保证结果的顺序与提交的顺序一致，如需一致可使用 stream 或 iter.Map。
type ResultPool[T any] struct {
	pool Pool
	agg  resultAggregator[T]
}

// Go 提交一个任务到协程池并择机执行。
func (p *ResultPool[T]) Go(f func() T) {
	p.pool.Go(func() {
		p.agg.add(f())
	})
}

// Wait 阻塞至所有任务全部完成。
func (p *ResultPool[T]) Wait() []T {
	p.pool.Wait()
	return p.agg.results
}

// MaxGoroutines 返回最大协程数。
func (p *ResultPool[T]) MaxGoroutines() int {
	return p.pool.MaxGoroutines()
}

// WithMaxGoroutines 设置最大协程数。默认为无限。
func (p *ResultPool[T]) WithMaxGoroutines(n int) *ResultPool[T] {
	p.pool.WithMaxGoroutines(n)
	return p
}

// WithErrors 转为能返回泛型结果但可能出错的多任务并发协程池。
func (p *ResultPool[T]) WithErrors() *ResultErrorPool[T] {
	return &ResultErrorPool[T]{
		errorPool: *p.pool.WithErrors(),
	}
}

// WithContext 转为任务带有上下文的并发协程池。
func (p *ResultPool[T]) WithContext(ctx context.Context) *ResultContextPool[T] {
	return &ResultContextPool[T]{
		contextPool: *p.pool.WithContext(ctx),
	}
}

type resultAggregator[T any] struct {
	mu      sync.Mutex
	results []T
}

func (r *resultAggregator[T]) add(res T) {
	r.mu.Lock()
	r.results = append(r.results, res)
	r.mu.Unlock()
}
