package errors

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

var (
	ErrConnectionClosed = errors.New("连接已关闭")
	ErrTimeout          = errors.New("超时")
	ErrWriteTimeout     = errors.New("写超时")
)

type ErrorType uint64

type Error struct {
	Err  error
	Type ErrorType
	Meta any
}

// 返回错误的消息字符串。
func (msg *Error) Error() string {
	return msg.Err.Error()
}

func (msg *Error) JSON() any {
	jsonData := make(map[string]any)
	if msg.Meta != nil {
		value := reflect.ValueOf(msg.Meta)
		switch value.Kind() {
		case reflect.Struct:
			return msg.Meta
		case reflect.Map:
			for _, key := range value.MapKeys() {
				jsonData[key.String()] = value.MapIndex(key).Interface()
			}
		default:
			jsonData["meta"] = msg.Meta
		}
	}
	if _, ok := jsonData["error"]; !ok {
		jsonData["error"] = msg.Error()
	}
	return jsonData
}

func (msg *Error) Unwrap() error {
	return msg.Err
}

func (msg *Error) IsType(flags ErrorType) bool {
	return (msg.Type & flags) > 0
}

func (msg *Error) SetType(flags ErrorType) *Error {
	msg.Type = flags
	return msg
}

func (msg *Error) SetMeta(data any) *Error {
	msg.Meta = data
	return msg
}

const (
	// ErrorTypeBind 用于 Context.Bind() 失败。
	ErrorTypeBind ErrorType = 1 << iota
	// ErrorTypeRender 用于 Context.Render() 失败。
	ErrorTypeRender
	// ErrorTypePrivate 表示一个私有的错误。
	ErrorTypePrivate
	// ErrorTypePublic 表示一个公开的错误。
	ErrorTypePublic
	// ErrorTypeAny 表示任何其他错误。
	ErrorTypeAny
)

var _ error = (*Error)(nil)

func New(err error, t ErrorType, meta any) *Error {
	return &Error{
		Err:  err,
		Type: t,
		Meta: meta,
	}
}

func NewPublic(err string) *Error {
	return New(errors.New(err), ErrorTypePublic, nil)
}

func NewPrivate(err string) *Error {
	return New(errors.New(err), ErrorTypePrivate, nil)
}

func Newf(t ErrorType, meta any, format string, v ...any) *Error {
	return New(fmt.Errorf(format, v...), t, meta)
}

func NewPublicf(format string, v ...any) *Error {
	return New(fmt.Errorf(format, v...), ErrorTypePublic, nil)
}

func NewPrivatef(format string, v ...any) *Error {
	return New(fmt.Errorf(format, v...), ErrorTypePrivate, nil)
}

type ErrorChain []*Error

func (c ErrorChain) String() string {
	if len(c) == 0 {
		return ""
	}
	var buf strings.Builder
	for i, msg := range c {
		fmt.Fprintf(&buf, "Error #%02d: %s\n", i+1, msg.Err)
		if msg.Meta != nil {
			fmt.Fprintf(&buf, "     Meta: %v\n", msg.Meta)
		}
	}
	return buf.String()
}

// Errors 返回错误的消息字符串切片。
func (c ErrorChain) Errors() []string {
	if len(c) == 0 {
		return nil
	}
	errorStrings := make([]string, len(c))
	for i, err := range c {
		errorStrings[i] = err.Error()
	}
	return errorStrings
}

// ByType 返回按指定类型过滤的错误数组。支持位或|操作。
func (c ErrorChain) ByType(t ErrorType) ErrorChain {
	if len(c) == 0 {
		return nil
	}
	if t == ErrorTypeAny {
		return c
	}
	var result ErrorChain
	for _, msg := range c {
		if msg.IsType(t) {
			result = append(result, msg)
		}
	}
	return result
}

// Last 返回错误切片中最后一个错误。
func (c ErrorChain) Last() *Error {
	if length := len(c); length > 0 {
		return c[length-1]
	}
	return nil
}

func (c ErrorChain) JSON() any {
	switch length := len(c); length {
	case 0:
		return nil
	case 1:
		return c.Last().JSON()
	default:
		jsonData := make([]any, length)
		for i, err := range c {
			jsonData[i] = err.JSON()
		}
		return jsonData
	}
}
