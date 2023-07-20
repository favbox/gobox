package syncx

import (
	"sync"

	"github.com/favbox/gobox/air/panics"
)

// NewStreamPool 创建新的流式并发协程池。
func NewStreamPool() *StreamPool {
	return &StreamPool{
		pool: *NewPool(),
	}
}

type (
	// 保证顺序一致的回调函数通道
	streamCallbackCh chan func()

	// StreamCallback 是流式任务的回调函数，将按提交顺序执行。
	StreamCallback func()
	// StreamTask 是按序执行的流式任务，完成后返回一个回调函数。
	StreamTask func() StreamCallback
	// StreamPool 是能保证顺序一致的多任务并发协程池，且并发数可控。
	StreamPool struct {
		pool       Pool
		callbackWg WaitGroup
		queue      chan streamCallbackCh // 回调通道的队列通道
		initOnce   sync.Once
	}
)

// Go 提交一个任务到协程池并择机执行。
func (p *StreamPool) Go(f StreamTask) {
	p.init()

	// 获取一条回调通道
	callbackCh := getCallbackCh()

	// 发往至回调通道
	p.queue <- callbackCh

	// 提交任务，择机执行
	p.pool.Go(func() {
		defer func() {
			// 如果任务出错，则发送一个空回调，以防回调运行器干等。
			if r := recover(); r != nil {
				callbackCh <- func() {}
				panic(r)
			}
		}()

		// 执行任务并将回调发往回调通道进行处理
		callback := f()
		callbackCh <- callback
	})
}

// Wait 阻塞至所有任务全部完成。
func (p *StreamPool) Wait() {
	p.init()

	// 确保即使任务出错发生恐慌，回调等待组也会执行回调运行程序。
	defer func() {
		close(p.queue)
		// 阻塞至回调全部完成。
		p.callbackWg.Wait()
	}()

	// 阻塞至任务全部完成。
	p.pool.Wait()
}

// WithMaxGoroutines 设置最大协程数。默认为无限。
func (p *StreamPool) WithMaxGoroutines(n int) *StreamPool {
	p.pool.WithMaxGoroutines(n)
	return p
}

// 确保流式协程池零值可用，并启动回调处理。
func (p *StreamPool) init() {
	p.initOnce.Do(func() {
		p.queue = make(chan streamCallbackCh, p.pool.MaxGoroutines()+1)

		// 执行回调。
		p.callbackWg.Go(p.callbackRunner)
	})
}

// 按任务提交的顺序执行回调函数。
// 仅一个运行实例。
func (p *StreamPool) callbackRunner() {
	var catcher panics.Catcher
	defer catcher.Repanic()

	// 从回调通道逐一读取回调函数并尝试执行
	for callbackCh := range p.queue {
		callback := <-callbackCh
		catcher.Try(callback)
		putCallbackCh(callbackCh)
	}
}

var streamCallbackChPool = sync.Pool{
	New: func() any {
		return make(streamCallbackCh, 1)
	},
}

// 从流式回调池中取一个回调通道。
func getCallbackCh() streamCallbackCh {
	return streamCallbackChPool.Get().(streamCallbackCh)
}

// 将一个可用回调通道放入回调池中。
func putCallbackCh(ch streamCallbackCh) {
	streamCallbackChPool.Put(ch)
}
