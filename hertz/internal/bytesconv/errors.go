package bytesconv

import "errors"

var (
	errEmptyInt            = errors.New("空整数")
	errUnexpectedFirstChar = errors.New("发现第一个字符异常，应为0-9")
)
