package syncx

import (
	"context"
	"sync"
)

// NewPool 创建新的并发协程池。
func NewPool() *Pool {
	return &Pool{}
}

// Pool 是用于多任务的并发协程池，且并发数可控。
//
// 用 Go() 提交一个任务，用 Wait() 阻塞并等待所有任务完成。
//
// 协程池启动和拆卸开销≈600纳秒，单任务开销≈300纳秒。故不建议用于短时任务。
type Pool struct {
	wg       WaitGroup
	limiter  limiter
	tasks    chan func()
	initOnce sync.Once
}

// Go 提交一个任务到协程池并择机执行。
func (p *Pool) Go(f func()) {
	p.init()

	if p.limiter == nil {
		// 不限并发
		select {
		case p.tasks <- f:
			// 有空闲协程可执行任务，无需派生新协程
		default:
			// 无空闲协程，需派生新协程
			p.wg.Go(func() { p.do(f) })
		}

	} else {
		// 限制并发
		select {
		case p.limiter <- struct{}{}:
			// 并发计数通道有空余，派生新协程并执行任务
			p.wg.Go(func() { p.do(f) })
		case p.tasks <- f:
			// 并发计数通道已满，将 f 加入任务通道以等待执行
			return
		}
	}
}

// Wait 阻塞至所有任务全部完成。
func (p *Pool) Wait() {
	// 确保任务通道已初始化
	p.init()

	// 关闭通道读写
	close(p.tasks)

	// 在任务全部完成后，重新初始化 initOnce
	defer func() { p.initOnce = sync.Once{} }()

	// 等待任务全部完成
	p.wg.Wait()
}

// MaxGoroutines 返回最大协程数。
func (p *Pool) MaxGoroutines() int {
	return p.limiter.cap()
}

// WithMaxGoroutines 设置最大协程数。默认为无限。
func (p *Pool) WithMaxGoroutines(n int) *Pool {
	p.panicIfInitialized()
	if n > 0 {
		p.limiter = make(limiter, n)
	}
	return p
}

// WithErrors 转为任务可能出错的并发协程池。
func (p *Pool) WithErrors() *ErrorPool {
	p.panicIfInitialized()
	return &ErrorPool{
		pool: p.deref(),
	}
}

// WithContext 转为任务带有上下文的并发协程池。
func (p *Pool) WithContext(ctx context.Context) *ContextPool {
	p.panicIfInitialized()
	ctx, cancel := context.WithCancel(ctx)
	return &ContextPool{
		errorPool: p.WithErrors().deref(),
		ctx:       ctx,
		cancel:    cancel,
	}
}

// 初始化协程池的任务通道，确保零值可用。
func (p *Pool) init() {
	p.initOnce.Do(func() {
		p.tasks = make(chan func())
	})
}

func (p *Pool) do(f func()) {
	// 释放一个并发计数
	defer p.limiter.release()

	// 先执行指定任务
	if f != nil {
		f()
	}

	// 再执行池中其他任务
	for fn := range p.tasks {
		fn()
	}
}

func (p *Pool) panicIfInitialized() {
	if p.tasks != nil {
		panic("在调用 Go() 之后，不可重新配置协程池。建议重新创建新池。")
	}
}

func (p *Pool) deref() Pool {
	p.panicIfInitialized()
	return Pool{limiter: p.limiter}
}

// 用于限制协程数量的通道。
type limiter chan struct{}

// 返回受限的协程容量。
func (l limiter) cap() int {
	return cap(l)
}

// 从受限通道中释放一个空位。
func (l limiter) release() {
	if l != nil {
		<-l
	}
}
