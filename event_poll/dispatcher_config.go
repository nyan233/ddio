package ddio

type DisPatcherConfig struct {
	// 连接处理器
	ConnHandler  ConnectionEventHandler
	// 负载均衡器
	Balanced     Balanced
	// 网络轮询器的配置
	EngineConfig *NetPollConfig
}
