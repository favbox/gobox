package panics

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"sync/atomic"
)

// Try 执行 f 并返回可能的恐慌。
func Try(f func()) *Recovered {
	var c Catcher
	c.Try(f)
	return c.Recovered()
}

// Catcher 用于捕获恐慌。您可以使用 Try 执行一个函数，
// 它将捕获任何引发的恐慌。Try 可由任意数量的协程调用任意次数。
// 一旦所有 Try 调用完成后，你可通过 Recovered() 获取第一个恐慌的值（如果有），也可以
// 通过 Repanic() 传播恐慌。
type Catcher struct {
	recovered atomic.Pointer[Recovered]
}

// Try 执行 f 并捕获恐慌，协程安全。
func (c *Catcher) Try(f func()) {
	defer c.tryRecover()
	f()
}

func (c *Catcher) tryRecover() {
	if val := recover(); val != nil {
		rp := NewRecovered(1, val)
		c.recovered.CompareAndSwap(nil, &rp)
	}
}

// Recovered 返回 Try 捕获到的第一个恐慌。
func (c *Catcher) Recovered() *Recovered {
	return c.recovered.Load()
}

// Repanic 如果 Try 捕获了恐慌，则重新触发。
func (c *Catcher) Repanic() {
	if val := c.Recovered(); val != nil {
		panic(val)
	}
}

// Recovered 是用 recover() 捕获到的恐慌。
type Recovered struct {
	// 恐慌的原始值
	Value any
	// 当恐慌恢复时，runtime.Callers 返回的调用方列表。
	// 可通过 runtime.CallersFrames 生成更详细的堆栈信息。
	Callers []uintptr
	// 从协程恐慌中恢复的格式化堆栈跟踪。
	Stack []byte
}

// String 返回字符串形式的恐慌原始值和堆栈。
func (p *Recovered) String() string {
	return fmt.Sprintf("%v\n\n错误堆栈：\n%s\n", p.Value, p.Stack)
}

// AsError 将恐慌转为带有堆栈的错误。
func (p *Recovered) AsError() error {
	if p == nil {
		return nil
	}
	return &ErrRecovered{*p}
}

var _ error = (*ErrRecovered)(nil)

// ErrRecovered 包装 Recovered 为实现 error 的结构体。
type ErrRecovered struct {
	Recovered
}

func (p *ErrRecovered) Error() string {
	return p.String()
}

func (p *ErrRecovered) Unwrap() error {
	if err, ok := p.Value.(error); ok {
		return err
	}
	return nil
}

// NewRecovered 从恐慌值和跟踪堆栈创建 Recovered。
// skip 参数允许调用者在收集堆栈跟踪跟踪时跳过指定堆栈帧。
// skip 为0意味着在堆栈跟踪中调用 NewRecovered。
func NewRecovered(skip int, value any) Recovered {
	// 64帧应该足够了
	var callers [64]uintptr
	n := runtime.Callers(skip+1, callers[:])
	return Recovered{
		Value:   value,
		Callers: callers[:n],
		Stack:   debug.Stack(),
	}
}
