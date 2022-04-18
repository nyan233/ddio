package ddio

import "net"

type NetPollConfig struct {
	Protocol int
	IP       net.IP
	Port     int
}

type ListenerConfig struct {
	ConnEHd       ConnectionEventHandler
	Balance       Balanced
	NetPollConfig *NetPollConfig
}

type EngineConfig struct {
	// 连接处理器
	ConnHandler ConnectionEventHandler
	// 负载均衡器的工厂函数
	NBalance NewBalance
	// 网络轮询器的配置
	NetPollConfig *NetPollConfig
}

type ConnConfig struct {
	// 触发OnData数据最多需要多少个Buffer Block
	// 一个Block大小为4KB
	OnDataNBlock int
}
