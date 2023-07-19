package protocol

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

type errorReader struct{}

func (er errorReader) Read(p []byte) (int, error) {
	return 0, fmt.Errorf("dummy")
}

func TestMultiForm(t *testing.T) {
	var r Request
	_, err := r.MultipartForm()
	fmt.Println(err)
}

func TestRequestBodyWriterWrite(t *testing.T) {
	w := requestBodyWriter{&Request{}}
	w.Write([]byte("test"))
	assert.Equal(t, "test", string(w.r.body.B))
}
