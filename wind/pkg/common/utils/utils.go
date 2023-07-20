package utils

import (
	"reflect"
	"runtime"

	"github.com/favbox/gobox/wind/internal/bytesconv"
)

// CaseInsensitiveCompare 不分大小写，比较两者是否相同。
// 比直接转小写后相比更快。
func CaseInsensitiveCompare(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}

	for i, n := 0, len(a); i < n; i++ {
		if a[i]|0x20 != b[i]|0x20 {
			return false
		}
	}
	return true
}

// NormalizeHeaderKey 规格化标头键名：将首字母及破折号后首字母转大写，其他转小写。
func NormalizeHeaderKey(b []byte, disableNormalizing bool) {
	if disableNormalizing {
		return
	}

	n := len(b)
	if n == 0 {
		return
	}

	// 首字母转大写
	b[0] = bytesconv.ToUpperTable[b[0]]

	// - 后面的字母转大写，其他字母转小写
	for i := 1; i < n; i++ {
		p := &b[i]
		if *p == '-' {
			i++
			if i < n {
				b[i] = bytesconv.ToUpperTable[b[i]]
			}
			continue
		}
		*p = bytesconv.ToLowerTable[*p]
	}
}

func Assert(guard bool, text string) {
	if !guard {
		panic(text)
	}
}

func NameOfFunction(f any) string {
	return runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
}
