package ext

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBufferSnippet(t *testing.T) {
	a := make([]byte, 39)
	b := make([]byte, 41)
	assert.NotContains(t, BufferSnippet(a), `"..."`)
	assert.Contains(t, BufferSnippet(b), `"..."`)
}

func TestRound2(t *testing.T) {
	fmt.Println(round2(2))
	fmt.Println(round2(3))
	fmt.Println(round2(10))
}
