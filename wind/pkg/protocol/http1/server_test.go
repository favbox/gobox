package http1

import (
	"sync"
	"testing"
)

var pool = &sync.Pool{
	New: func() any {
		return &eventStack{}
	},
}

func TestTraceEventCompleted(t *testing.T) {
	server := &Server{}
	server.eventStackPool = pool
	server.EnableTrace = true
}
