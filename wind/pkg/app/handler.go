package app

import (
	"context"
	"reflect"
)

// HandlerFunc 是请求的处理函数。
type HandlerFunc func(c context.Context, ctx *RequestContext)

// HandlerChain 处理链，定义了一组请求处理函数。
type HandlerChain []HandlerFunc

// Last 返回处理链的最后一个处理函数。
//
// 通常来说，最后一个处理函数是主处理函数。
func (c HandlerChain) Last() HandlerFunc {
	if length := len(c); length > 0 {
		return c[length-1]
	}
	return nil
}

var handlerNames = make(map[uintptr]string)

func SetHandlerName(handler HandlerFunc, name string) {
	handlerNames[getFuncAddr(handler)] = name
}

func GetHandlerName(handler HandlerFunc) string {
	return handlerNames[getFuncAddr(handler)]
}

func getFuncAddr(v any) uintptr {
	return reflect.ValueOf(reflect.ValueOf(v)).Field(1).Pointer()
}
