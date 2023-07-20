package syncx

import "context"

// ResultContextPool 是带有上下文并返回泛型结果但可能出错的多任务并发协程池。
//
// 用 Go() 提交一个任务，用 Wait() 阻塞并等待所有任务完成。
type ResultContextPool[T any] struct {
	contextPool    ContextPool
	agg            resultAggregator[T]
	collectErrored bool // 是否收集出错任务的结果
}

// Go 提交一个任务到协程池并择机执行。
func (p *ResultContextPool[T]) Go(f func(ctx context.Context) (T, error)) {
	p.contextPool.Go(func(ctx context.Context) error {
		res, err := f(ctx)
		if err == nil || p.collectErrored {
			p.agg.add(res)
		}
		return err
	})
}

// Wait 阻塞至所有任务全部完成。
func (p *ResultContextPool[T]) Wait() ([]T, error) {
	err := p.contextPool.Wait()
	return p.agg.results, err
}

// WithMaxGoroutines 设置最大协程数。默认为无限。
func (p *ResultContextPool[T]) WithMaxGoroutines(n int) *ResultContextPool[T] {
	p.contextPool.WithMaxGoroutines(n)
	return p
}

// WithFirstError 设置为仅返回第一个错误，默认返回全部错误。
func (p *ResultContextPool[T]) WithFirstError() *ResultContextPool[T] {
	p.contextPool.WithFirstError()
	return p
}

// WithCollectErrored 配置为即使任务出错也收集结果。
//
// 默认忽略出错任务的结果，仅收集错误。
func (p *ResultContextPool[T]) WithCollectErrored() *ResultContextPool[T] {
	p.collectErrored = true
	return p
}

// WithCancelOnError 设置第一次出错应取消所有协程。
func (p *ResultContextPool[T]) WithCancelOnError() *ResultContextPool[T] {
	p.contextPool.WithCancelOnError()
	return p
}
