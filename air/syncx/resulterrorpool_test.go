package syncx

import (
	"errors"
	"testing"
	"time"

	"github.com/favbox/gobox/wind/pkg/common/test/assert"
)

func TestResultErrorGroup(t *testing.T) {
	t.Parallel()

	err1 := errors.New("err1")
	err2 := errors.New("err2")

	t.Run("任务没错，返回结果也没错", func(t *testing.T) {
		t.Parallel()

		p := NewResultPool[int]().WithErrors()
		p.Go(func() (int, error) { return 1, nil })
		res, err := p.Wait()
		assert.Nil(t, err)
		assert.DeepEqual(t, []int{1}, res)
	})

	t.Run("默认 —— 任务出错，返回错误不返回结果", func(t *testing.T) {
		t.Parallel()

		p := NewResultPool[int]().WithErrors()
		p.Go(func() (int, error) { return 0, err1 })
		res, err := p.Wait()
		assert.True(t, len(res) == 0)
		assert.True(t, errors.Is(err, err1))
	})

	t.Run("配置 —— 任务出错，返回错误也返回结果", func(t *testing.T) {
		t.Parallel()

		p := NewResultPool[int]().WithErrors().WithCollectErrored()
		p.Go(func() (int, error) { return 0, err1 })
		res, err := p.Wait()
		assert.True(t, len(res) == 1) // 出错结果也被采集到了
		assert.True(t, errors.Is(err, err1))
	})

	t.Run("只返回第一个错误", func(t *testing.T) {
		p := NewResultPool[int]().WithErrors().WithFirstError()
		sync := make(chan struct{})
		p.Go(func() (int, error) {
			<-sync
			time.Sleep(100 * time.Millisecond)
			return 0, err1
		})
		p.Go(func() (int, error) {
			defer close(sync)
			return 0, err2
		})
		res, err := p.Wait()
		assert.True(t, len(res) == 0)
		assert.True(t, errors.Is(err, err2))
		assert.False(t, errors.Is(err, err1))
	})
}
