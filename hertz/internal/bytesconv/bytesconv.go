package bytesconv

import (
	"net/http"
	"reflect"
	"time"
	"unsafe"
)

const (
	upperHex = "0123456789ABCDEF"
	lowerHex = "0123456789abcdef"
)

func LowercaseBytes(b []byte) {
	for i, n := 0, len(b); i < n; i++ {
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

// AppendQuotedArg 将源参数切片转义后附加到目标并返回。等效于 url.QuoteEscape。
func AppendQuotedArg(dst, src []byte) []byte {
	for _, c := range src {
		switch {
		case c == ' ':
			dst = append(dst, '+')
		case QuotedArgShouldEscapeTable[int(c)] != 0:
			dst = append(dst, '%', upperHex[c>>4], upperHex[c&0xf])
		default:
			dst = append(dst, c)
		}
	}
	return dst
}

// AppendQuotedPath 将源路径切片转义后附加到目标并返回。等效于 url.EscapedPath。
func AppendQuotedPath(dst, src []byte) []byte {
	// 修复该问题 https://github.com/golang/go/issues/11202
	if len(src) == 1 && src[0] == '*' {
		return append(dst, '*')
	}

	for _, c := range src {
		if QuotedPathShouldEscapeTable[int(c)] != 0 {
			dst = append(dst, '%', upperHex[c>>4], upperHex[c&15])
		} else {
			dst = append(dst, c)
		}
	}
	return dst
}

// AppendUint 附加正整数 n 到字节切片 dst 并返回。
func AppendUint(dst []byte, n int) []byte {
	if n < 0 {
		panic("BUG：int 必须为正数")
	}

	var b [20]byte
	buf := b[:]
	i := len(buf)
	var q int
	for n >= 10 {
		i--
		q = n / 10
		buf[i] = '0' + byte(n-q*10)
		n = q
	}
	i--
	buf[i] = '0' + byte(n)

	dst = append(dst, buf[i:]...)
	return dst
}

// AppendHTTPDate 附加 HTTP 兼容的时间表示比到字节切片 dst 并返回。
func AppendHTTPDate(dst []byte, date time.Time) []byte {
	return date.UTC().AppendFormat(dst, http.TimeFormat)
}

// ParseUintBuf 从字节缓冲区中解析出 uint。
func ParseUintBuf(b []byte) (v int, n int, err error) {
	n = len(b)
	if n == 0 {
		return -1, 0, errEmptyInt
	}
	for i := 0; i < n; i++ {
		c := b[i]
		k := c - '0'
		if k > 9 {
			if i == 0 {
				return -1, i, errUnexpectedFirstChar
			}
			return v, i, nil
		}
		vNew := 10*v + int(k)
		// 测试溢出
		if vNew < v {
			return -1, i, errTooLongInt
		}
		v = vNew
	}
	return
}

// ParseUint 从字节切片中解析出 uint。
func ParseUint(buf []byte) (int, error) {
	v, n, err := ParseUintBuf(buf)
	if n != len(buf) {
		return -1, errUnexpectedTrailingChar
	}
	return v, err
}
