package utils

import (
	"strings"
)

// AddMissingPort 若 addr 中没有端口的话，则 TLS 添加 :443，非 TLS 添加 :80。
// 主机端口中的 IPv6 地址必须用括号括起来，如 "[::1]:80", "[::1%lo0]:80"。
func AddMissingPort(addr string, isTLS bool) string {
	if strings.IndexByte(addr, ':') >= 0 {
		endOfV6 := strings.IndexByte(addr, ']')
		// 我们不关心地址的有效性，只需检查“]”之后是否有更多字节
		if endOfV6 < len(addr)-1 {
			return addr
		}
	}
	if !isTLS {
		return addr + ":80"
	}
	return addr + ":443"
}
