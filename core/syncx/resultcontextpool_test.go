package syncx

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestResultContextPool(t *testing.T) {
	t.Parallel()

	err1 := errors.New("err1")

	t.Run("before wait", func(t *testing.T) {
		t.Parallel()
		g := NewResultPool[int]().WithContext(context.Background())
		g.Go(func(context.Context) (int, error) { return 0, nil })
		require.Panics(t, func() { g.WithMaxGoroutines(10) })
	})

	t.Run("出错就退出", func(t *testing.T) {
		t.Parallel()
		p := NewResultPool[int]().WithContext(context.Background()).WithCancelOnError()
		p.Go(func(ctx context.Context) (int, error) {
			fmt.Println("到这里了吗1")
			<-ctx.Done()
			return 0, ctx.Err()
		})
		p.Go(func(ctx context.Context) (int, error) {
			fmt.Println("到这里了吗2")
			return 0, err1
		})
		res, err := p.Wait()
		require.Len(t, res, 0)
		require.ErrorIs(t, err, context.Canceled)
		require.ErrorIs(t, err, err1)
	})

	t.Run("出错不退出", func(t *testing.T) {
		t.Parallel()
		p := NewResultPool[int]().WithContext(context.Background())
		p.Go(func(ctx context.Context) (int, error) {
			select {
			case <-ctx.Done():
				fmt.Println("到这里了吗1")
				return 0, ctx.Err()
			case <-time.After(10 * time.Millisecond):
				fmt.Println("毫秒好快啊")
				return 0, nil
			}
		})
		p.Go(func(ctx context.Context) (int, error) {
			fmt.Println("到这里了吗2")
			return 0, err1
		})
		res, err := p.Wait()
		require.Len(t, res, 1)
		require.NotErrorIs(t, err, context.Canceled)
		require.ErrorIs(t, err, err1)
	})
}
