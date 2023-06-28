package syncx

import "context"

// ResultErrorPool 是能返回泛型结果但可能出错的多任务并发协程池。
//
// 用 Go() 提交一个任务，用 Wait() 阻塞并等待所有任务完成。
type ResultErrorPool[T any] struct {
	errorPool      ErrorPool
	agg            resultAggregator[T]
	collectErrored bool // 是否收集出错任务的结果
}

// Go 提交一个任务到协程池并择机执行。
func (p *ResultErrorPool[T]) Go(f func() (T, error)) {
	p.errorPool.Go(func() error {
		res, err := f()
		if err == nil || p.collectErrored {
			p.agg.add(res)
		}
		return err
	})
}

// Wait 阻塞至所有任务全部完成。
func (p *ResultErrorPool[T]) Wait() ([]T, error) {
	err := p.errorPool.Wait()
	return p.agg.results, err
}

// WithContext 转为任务带有上下文的并发协程池，以便任务出错全体取消。
func (p *ResultErrorPool[T]) WithContext(ctx context.Context) *ResultContextPool[T] {
	return &ResultContextPool[T]{
		contextPool: *p.errorPool.WithContext(ctx),
	}
}

// WithMaxGoroutines 设置最大协程数。默认为无限。
func (p *ResultErrorPool[T]) WithMaxGoroutines(n int) *ResultErrorPool[T] {
	p.errorPool.WithMaxGoroutines(n)
	return p
}

// WithFirstError 设置为仅返回第一个错误，默认返回全部错误。
func (p *ResultErrorPool[T]) WithFirstError() *ResultErrorPool[T] {
	p.errorPool.WithFirstError()
	return p
}

// WithCollectErrored 配置为即使任务出错也收集结果。
//
// 默认忽略出错任务的结果，仅收集错误。
func (p *ResultErrorPool[T]) WithCollectErrored() *ResultErrorPool[T] {
	p.collectErrored = true
	return p
}
