package ext

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBufferSnippet(t *testing.T) {
	a := make([]byte, 39)
	b := make([]byte, 41)
	assert.NotContains(t, BufferSnippet(a), `"..."`)
	assert.Contains(t, BufferSnippet(b), `"..."`)
}
