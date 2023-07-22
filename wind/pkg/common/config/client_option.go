package config

type ConnPoolState struct {
	// 连接池的连接数量。这些连接为空闲连接。
	PoolConnNum int
	// 连接总数。
	TotalConnNum int
	// 挂起的连接数量。
	WaitConnNum int
	// HostClient 地址
	Add string
}

type HostClientState interface {
	ConnPoolState() ConnPoolState
}

type HostClientStateFunc func(HostClientState)
