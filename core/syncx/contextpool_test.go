package syncx

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/favbox/gobox/http/pkg/common/test/assert"
)

func ExampleContextPool() {
	p := NewPool().
		WithMaxGoroutines(4).
		WithContext(context.Background()).
		WithCancelOnError()

	for i := 0; i < 3; i++ {
		i := i
		p.Go(func(ctx context.Context) error {
			if i == 2 {
				return errors.New("我错啦，全体退出！")
			}
			<-ctx.Done()
			return nil
		})
	}

	err := p.Wait()
	fmt.Println(err)
	// Output:
	// 我错啦，全体退出！
}

func TestContextPool(t *testing.T) {
	t.Parallel()

	bgCtx := context.Background()
	err1 := errors.New("错误1")
	err2 := errors.New("错误2")

	t.Run("可以取消的并发任务", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(bgCtx)
		p := NewPool().WithContext(ctx)
		p.Go(func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		})
		// 在需要主动取消了
		cancel()
		assert.True(t, p.Wait().Error() == context.Canceled.Error())
	})

	t.Run("超时报错的并发任务", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithTimeout(bgCtx, time.Millisecond)
		defer cancel()
		p := NewPool().WithContext(ctx)
		p.Go(func(ctx context.Context) error {
			// 强制到期
			<-ctx.Done()
			return ctx.Err()
		})
		assert.True(t, p.Wait().Error() == context.DeadlineExceeded.Error())
	})

	t.Run("超时前就退出的并发任务", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithTimeout(bgCtx, time.Millisecond)
		defer cancel()
		p := NewPool().WithContext(ctx)
		p.Go(func(ctx context.Context) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Microsecond):
				return nil
			}
		})
		assert.Nil(t, p.Wait())
	})

	t.Run("未配置 WithCancelOnError", func(t *testing.T) {
		t.Parallel()

		p := NewPool().WithContext(bgCtx)
		p.Go(func(ctx context.Context) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(10 * time.Millisecond):
				return nil
			}
		})
		p.Go(func(ctx context.Context) error {
			return err1
		})

		err := p.Wait()

		assert.False(t, errors.Is(err, context.Canceled))
		assert.True(t, errors.Is(err, err1))
	})

	t.Run("已配置 WithCancelOnError", func(t *testing.T) {
		t.Parallel()

		p := NewPool().WithContext(bgCtx).WithCancelOnError()
		p.Go(func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		})
		p.Go(func(ctx context.Context) error {
			return err1
		})

		err := p.Wait()

		assert.True(t, errors.Is(err, context.Canceled))
		assert.True(t, errors.Is(err, err1))
	})

	t.Run("只要第一个错误", func(t *testing.T) {
		t.Parallel()

		p := NewPool().WithContext(bgCtx).WithFirstError()
		sync := make(chan struct{})
		p.Go(func(ctx context.Context) error {
			defer close(sync)
			return err1
		})
		p.Go(func(ctx context.Context) error {
			<-sync
			time.Sleep(10 * time.Millisecond)
			return err2
		})
		err := p.Wait()
		fmt.Println(err)
		//assert.True(t, errors.Is(err, err1))
		//assert.False(t, errors.Is(err, err2))
	})
}
