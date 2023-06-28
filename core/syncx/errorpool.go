package syncx

import (
	"context"
	"errors"
	"sync"
)

// ErrorPool 是用于任务可能出错的并发协程池。
type ErrorPool struct {
	pool           Pool
	onlyFirstError bool
	mu             sync.Mutex
	errs           error
}

// Go 提交一个任务到协程池并择机执行。
func (p *ErrorPool) Go(f func() error) {
	p.pool.Go(func() {
		p.addErr(f())
	})
}

// Wait 阻塞至所有任务全部完成。
func (p *ErrorPool) Wait() error {
	p.pool.Wait()
	return p.errs
}

// WithMaxGoroutines 设置最大协程数。默认为无限。
func (p *ErrorPool) WithMaxGoroutines(n int) *ErrorPool {
	p.pool.panicIfInitialized()
	p.pool.WithMaxGoroutines(n)
	return p
}

// WithFirstError 设置为仅返回第一个错误，默认返回全部错误。
func (p *ErrorPool) WithFirstError() *ErrorPool {
	p.pool.panicIfInitialized()
	p.onlyFirstError = true
	return p
}

// WithContext 转为任务带有上下文的并发协程池，以便任务出错全体取消。
func (p *ErrorPool) WithContext(ctx context.Context) *ContextPool {
	p.pool.panicIfInitialized()
	ctx, cancel := context.WithCancel(ctx)
	return &ContextPool{
		errorPool: p.deref(),
		ctx:       ctx,
		cancel:    cancel,
	}
}

func (p *ErrorPool) addErr(err error) {
	if err != nil {
		p.mu.Lock()
		defer p.mu.Unlock()

		if p.onlyFirstError {
			if p.errs == nil {
				p.errs = err
			}
		} else {
			p.errs = errors.Join(p.errs, err)
		}
	}
}

func (p *ErrorPool) deref() ErrorPool {
	return ErrorPool{
		pool:           p.pool.deref(),
		onlyFirstError: p.onlyFirstError,
	}
}
