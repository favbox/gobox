package syncx

import (
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/favbox/gobox/wind/pkg/common/test/assert"
)

func ExampleWaitGroup_Wait() {
	var count atomic.Int64

	var wg WaitGroup
	for i := 0; i < 10; i++ {
		wg.Go(func() { count.Add(1) })
	}
	wg.Wait()

	fmt.Println(count.Load())
	// Output:
	// 10
}

func ExampleWaitGroup_WaitAndRecover() {
	var wg WaitGroup
	wg.Go(func() { panic("出错啦") })
	r := wg.WaitAndRecover()

	fmt.Println(r.Value)
	// Output:
	// 出错啦
}

func TestWaitGroup(t *testing.T) {
	t.Parallel()

	t.Run("恐慌会被捕获", func(t *testing.T) {
		t.Parallel()

		var wg WaitGroup
		wg.Go(func() { panic("出错啦") })

		assert.Panic(t, wg.Wait)
	})

	t.Run("第一个被捕获的恐慌会被返回", func(t *testing.T) {
		t.Parallel()

		var wg WaitGroup
		wg.Go(func() { panic("1出错啦") })
		wg.Go(func() { panic("2出错啦") })
		r := wg.WaitAndRecover()

		assert.NotNil(t, r)
	})

	t.Run("恐慌的协程不会影响正常协程的执行", func(t *testing.T) {
		t.Parallel()
		var count atomic.Int64

		var wg WaitGroup
		wg.Go(func() { panic("1出错啦") })
		wg.Go(func() { panic("2出错啦") })
		wg.Go(func() { count.Add(1) })
		_ = wg.WaitAndRecover()

		assert.True(t, count.Load() == int64(1))
	})
}
