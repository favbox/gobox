package registry

import "net"

const DefaultWeight = 10

var NoopRegistry Registry = &noopRegistry{}

// Registry 是服务注册的扩展接口。
type Registry interface {
	Register(info *Info) error
	Deregister(info *Info) error
}

// Info 用于服务注册的信息。
type Info struct {
	ServiceName string            // 在 wind 中会被默认设置
	Addr        net.Addr          // 在 wind 中会被默认设置
	Weight      int               // 在 wind 中会被默认设置
	Tags        map[string]string // 其他扩展信息
}

// 无操作的服务注册实现结构体
type noopRegistry struct{}

func (n noopRegistry) Register(info *Info) error {
	return nil
}

func (n noopRegistry) Deregister(info *Info) error {
	return nil
}
