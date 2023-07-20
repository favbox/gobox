package hack

import (
	"reflect"
	"unsafe"
)

// StringToBytes 字符串转字节切片（零内存分配）。
//
// 这是一个浅拷贝，即返回的字节切片复用字符串的底层数组，故
// 在任何场景下你都不要修改返回的字节切片。
func StringToBytes(s string) (b []byte) {
	bh := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	sh := (*reflect.StringHeader)(unsafe.Pointer(&s))
	bh.Data = sh.Data
	bh.Len = sh.Len
	bh.Cap = sh.Len
	return b
}

// BytesToString 字节切片转字符串（零内存分配）。
//
// 这是一个浅拷贝，即返回的字符串复用字节切片的底层数组，故
// 在任何场景下你都不要修改返回的字符串。
func BytesToString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}
