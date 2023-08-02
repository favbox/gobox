package registry

import (
	"testing"

	"github.com/favbox/gosky/wind/pkg/common/test/assert"
)

func TestNoopRegistry(t *testing.T) {
	reg := noopRegistry{}
	assert.Nil(t, reg.Register(&Info{}))
	assert.Nil(t, reg.Deregister(&Info{}))
}
