package bytesconv

import "errors"

var (
	errEmptyInt               = errors.New("整数为空")
	errUnexpectedFirstChar    = errors.New("发现第一个字符异常，应为0-9")
	errUnexpectedTrailingChar = errors.New("发现第尾随字符异常，应为0-9")
	errTooLongInt             = errors.New("int过长")
)
