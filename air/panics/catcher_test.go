package panics

import (
	"errors"
	"strings"
	"testing"

	"github.com/favbox/gobox/wind/pkg/common/test/assert"
)

func TestCatcher(t *testing.T) {
	t.Parallel()

	var pc Catcher

	i := 0
	pc.Try(func() { i += 1 })
	pc.Try(func() { panic("模拟恐慌") })
	pc.Try(func() { i += 1 })

	r := pc.Recovered()
	assert.True(t, i == 2)
	assert.True(t, r.Value == "模拟恐慌")
	assert.Panic(t, pc.Repanic)
	assert.NotNil(t, r.AsError())
}

func TestTry(t *testing.T) {
	t.Parallel()

	err := errors.New("这是一个错误")

	r := Try(func() { panic(err) })
	assert.True(t, strings.Contains(r.String(), err.Error()))
}
