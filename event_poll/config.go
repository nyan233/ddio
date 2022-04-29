package ddio

import "net"

type NetPollConfig struct {
	// 程序监听了多个地址
	IsMultiAddr bool
	Protocol    int
	IP          net.IP
	Port        int
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
	// 绑定的地址
	// Protocol://ip:port?level=n
	// level设定地址的优先级，优先级越高为该地址分配的监听线程和处理线程就越多
	// Level总量是10,设置的数量是为该地址分配的监听线程的百分比，最少为1个
	//   Example: tcp://127.0.0.1:8080?level=5
	MultiAddr []string
}

type ConnConfig struct {
	// 触发OnData数据最多需要多少个Buffer Block
	// 一个Block大小为4KB
	OnDataNBlock int
	// 尝试Non-Block read()的最大次数
	MaxReadSysCallNumber int
	// 尝试Non-Block write()的最大次数
	MaxWriteSysCallNumber int
}
