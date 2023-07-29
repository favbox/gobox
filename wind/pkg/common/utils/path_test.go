package utils

import (
	"testing"

	"github.com/favbox/gosky/wind/pkg/common/test/assert"
)

// 函数 AddMissingPort 只添加丢失的端口，不考虑其他错误情况。
func TestPathAddMissingPort(t *testing.T) {
	ipList := []string{"127.0.0.1", "111.111.1.1", "[0:0:0:0:0:ffff:192.1.56.10]", "[0:0:0:0:0:ffff:c0a8:101]", "www.foobar.com"}
	for _, ip := range ipList {
		assert.DeepEqual(t, ip+":443", AddMissingPort(ip, true))
		assert.DeepEqual(t, ip+":80", AddMissingPort(ip, false))
		customizedPort := ":8080"
		assert.DeepEqual(t, ip+customizedPort, AddMissingPort(ip+customizedPort, true))
		assert.DeepEqual(t, ip+customizedPort, AddMissingPort(ip+customizedPort, false))
	}
}
