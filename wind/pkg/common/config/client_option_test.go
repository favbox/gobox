package config

import (
	"testing"
	"time"

	"github.com/favbox/gosky/wind/pkg/common/test/assert"
	"github.com/favbox/gosky/wind/pkg/protocol/consts"
)

// TestDefaultClientOptions 测试客户端选项的默认值
func TestDefaultClientOptions(t *testing.T) {
	options := NewClientOptions([]ClientOption{})

	assert.DeepEqual(t, consts.DefaultDialTimeout, options.DialTimeout)
	assert.DeepEqual(t, consts.DefaultMaxConnsPerHost, options.MaxConnsPerHost)
	assert.DeepEqual(t, consts.DefaultMaxIdleConnDuration, options.MaxIdleConnDuration)
	assert.DeepEqual(t, true, options.KeepAlive)
}

// TestCustomClientOptions 测试客户端选项的自定义值。
func TestCustomClientOptions(t *testing.T) {
	options := NewClientOptions([]ClientOption{})

	options.Apply([]ClientOption{
		{
			F: func(o *ClientOptions) {
				o.DialTimeout = 2 * time.Second
			},
		},
	})
	assert.DeepEqual(t, 2*time.Second, options.DialTimeout)
}
