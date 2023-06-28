package bytesconv

import (
	"reflect"
	"unsafe"
)

func LowercaseBytes(b []byte) {
	for i := 0; i < len(b); i++ {
		p := &b[i]
		*p = ToLowerTable[*p]
	}
}

// B2s 将字节切片转为字符串，且不分配内存。
// 详见 https://groups.google.com/forum/#!msg/Golang-Nuts/ENgbUzYvCuU/90yGx7GUAgAJ 。
//
// 注意：如果字符串或切片的标头在未来的go版本中更改，该方法可能会出错。
func B2s(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

// S2b 将字符串转为字节切片，且不分配内存。
//
// 注意：如果字符串或切片的标头在未来的go版本中更改，该方法可能会出错。
func S2b(s string) (b []byte) {
	bh := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	sh := (*reflect.StringHeader)(unsafe.Pointer(&s))
	bh.Data = sh.Data
	bh.Len = sh.Len
	bh.Cap = sh.Len
	return b
}
