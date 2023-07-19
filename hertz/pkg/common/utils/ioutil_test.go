package utils

import (
	"bytes"
	"testing"

	"github.com/favbox/gobox/hertz/pkg/network"
	"github.com/stretchr/testify/assert"
)

func TestIOUtilCopyBuffer(t *testing.T) {
	var writeBuffer bytes.Buffer
	str := "hertz is very good!!!"
	src := bytes.NewBufferString(str)
	dst := network.NewWriter(&writeBuffer)
	var buf []byte
	// src.Len() will change, when use src.read(p []byte)
	srcLen := int64(src.Len())
	written, err := CopyBuffer(dst, src, buf)

	assert.Equal(t, written, srcLen)
	assert.Equal(t, err, nil)
	assert.Equal(t, []byte(str), writeBuffer.Bytes())
}

func TestIOUtilCopyBufferWithNilBuffer(t *testing.T) {
	var writeBuffer bytes.Buffer
	str := "hertz is very good!!!"
	src := bytes.NewBufferString(str)
	dst := network.NewWriter(&writeBuffer)
	// src.Len() will change, when use src.read(p []byte)
	srcLen := int64(src.Len())
	written, err := CopyBuffer(dst, src, nil)

	assert.Equal(t, written, srcLen)
	assert.Equal(t, err, nil)
	assert.Equal(t, []byte(str), writeBuffer.Bytes())
}

func TestIoutilCopyZeroAlloc(t *testing.T) {
	var writeBuffer bytes.Buffer
	str := "hertz is very good!!!"
	src := bytes.NewBufferString(str)
	dst := network.NewWriter(&writeBuffer)
	srcLen := int64(src.Len())
	written, err := CopyZeroAlloc(dst, src)

	assert.Equal(t, written, srcLen)
	assert.Equal(t, err, nil)
	assert.Equal(t, []byte(str), writeBuffer.Bytes())
}
