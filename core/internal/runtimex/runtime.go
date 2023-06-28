package runtimex

import _ "unsafe" // 用于连接名称

//go:linkname Fastrand runtime.fastrand
func Fastrand() uint32
