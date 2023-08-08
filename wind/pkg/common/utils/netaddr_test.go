package utils

import (
	"testing"

	"github.com/favbox/gosky/wind/pkg/common/test/assert"
)

func TestNetAddr(t *testing.T) {
	networkAddr := NewNetAddr("tcp", "127.0.0.1")

	assert.DeepEqual(t, networkAddr.Network(), "tcp")
	assert.DeepEqual(t, networkAddr.String(), "127.0.0.1")
}
