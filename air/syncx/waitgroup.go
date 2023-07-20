package syncx

import (
	"sync"

	"github.com/favbox/gobox/air/panics"
)

// NewWaitGroup 创建新的协程等待组。
func NewWaitGroup() *WaitGroup {
	return &WaitGroup{}
}

// WaitGroup 是 sync.WaitGroup 的升级版。
//
// 可捕获恐慌并回传给 Wait() 的调用者。
type WaitGroup struct {
	waitGroup sync.WaitGroup
	catcher   panics.Catcher
}

// Go 在新的子协程中执行 f。
//
// Go 能自动加减 sync.WaitGroup 的计数器。
func (w *WaitGroup) Go(f func()) {
	w.waitGroup.Add(1)
	go func() {
		defer w.waitGroup.Done()
		w.catcher.Try(f)
	}()
}

// Wait 阻塞至任务全部完成，出错会 panic。
func (w *WaitGroup) Wait() {
	// 等待任务全部完成。
	w.waitGroup.Wait()
	// 如有恐慌，传播恐慌。
	w.catcher.Repanic()
}

// WaitAndRecover 阻塞至任务全部完成，出错不传播恐慌而是返回恐慌。
func (w *WaitGroup) WaitAndRecover() *panics.Recovered {
	// 等待任务全部完成。
	w.waitGroup.Wait()
	// 如有恐慌，重新触发。
	return w.catcher.Recovered()
}
