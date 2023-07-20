package syncx

import (
	"errors"
	"fmt"
	"testing"

	"github.com/favbox/gobox/wind/pkg/common/test/assert"
)

func ExampleErrorPool() {
	p := NewPool().WithErrors()
	for i := 0; i < 3; i++ {
		i := i
		p.Go(func() error {
			if i == 2 {
				return errors.New("有个任务出错啦")
			}
			return nil
		})
	}
	err := p.Wait()
	fmt.Println(err)
	// Output:
	// 有个任务出错啦
}

func TestErrorPool(t *testing.T) {
	t.Parallel()

	err1 := errors.New("错误1")
	err2 := errors.New("错误2")

	t.Run("默认返回所有错误", func(t *testing.T) {
		p := NewPool().WithErrors()
		p.Go(func() error { return err1 })
		p.Go(func() error { return nil })
		p.Go(func() error { return err2 })
		err := p.Wait()
		assert.True(t, errors.Is(err, err1))
		assert.True(t, errors.Is(err, err2))
	})

	t.Run("如有恐慌，传播恐慌", func(t *testing.T) {
		p := NewPool().WithErrors()
		p.Go(func() error { return err1 })
		p.Go(func() error { panic("地震啦") })
		p.Go(func() error { return err2 })
		assert.Panic(t, func() { _ = p.Wait() })
	})
}
