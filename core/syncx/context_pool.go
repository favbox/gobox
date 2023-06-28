package syncx

import "context"

// ContextPool 是用于任务带有上下文的并发协程池。
type ContextPool struct {
	errorPool     ErrorPool
	ctx           context.Context
	cancel        context.CancelFunc
	cancelOnError bool
}

// Go 提交一个任务到协程池并择机执行。
func (p *ContextPool) Go(f func(ctx context.Context) error) {
	p.errorPool.Go(func() error {
		if p.cancelOnError {
			// 执行后，遇错退出
			defer func() {
				if r := recover(); r != nil {
					p.cancel()
					panic(r)
				}
			}()
		}

		err := f(p.ctx)
		if err != nil && p.cancelOnError {
			// 先添加错误，再退出
			p.errorPool.addErr(err)
			p.cancel()
			return nil
		}
		return err
	})
}

// Wait 阻塞至所有任务全部完成。
func (p *ContextPool) Wait() error {
	return p.errorPool.Wait()
}

// WithMaxGoroutines 设置最大协程数。默认为无限。
func (p *ContextPool) WithMaxGoroutines(n int) *ContextPool {
	p.errorPool.WithMaxGoroutines(n)
	return p
}

// WithCancelOnError 设置第一次出错应取消所有协程。
func (p *ContextPool) WithCancelOnError() *ContextPool {
	p.cancelOnError = true
	return p
}

// WithFirstError 设置为仅返回第一个错误。默认返回全部错误。
func (p *ContextPool) WithFirstError() *ContextPool {
	p.errorPool.WithFirstError()
	return p
}
