// Package timer 该包源自 fasthttp v1.36.0
//
// 提供计时器。
package timer

import (
	"sync"
	"time"
)

var timerPool sync.Pool

// AcquireTimer 从池中返回一个计时器，并在超时后更新它以便在其通道上发送当前时间。
//
// 当返回的计时器不用时，可通过 ReleaseTimer 放回池中，以减少 GC 负载。
func AcquireTimer(timeout time.Duration) *time.Timer {
	v := timerPool.Get()
	if v == nil {
		return time.NewTimer(timeout)
	}
	t := v.(*time.Timer)
	initTimer(t, timeout)
	return t
}

// ReleaseTimer 将通过 AcquireTimer 获取的计时器返回池中，并阻止计时器启动。
//
// 别访问已释放的计时器，或读取其通道，否则可能造成数据竞赛。
func ReleaseTimer(t *time.Timer) {
	stopTimer(t)
	timerPool.Put(t)
}

func initTimer(t *time.Timer, timeout time.Duration) *time.Timer {
	if t == nil {
		return time.NewTimer(timeout)
	}
	if t.Reset(timeout) {
		panic("BUG: 活动计时器被困在 initTimer() 中了")
	}
	return t
}

func stopTimer(t *time.Timer) {
	if !t.Stop() {
		// 如果计时器已停止但无人收集其值，则从其通道收集可能添加的时间。
		select {
		case <-t.C:
		default:
		}
	}
}
