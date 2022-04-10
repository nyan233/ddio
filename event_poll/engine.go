package ddio

// Engine 实例
type Engine struct {
	// 引擎绑定的多个监听多路事件派发器
	mds []*ListenerMultiEventDispatcher
	// 配置参数
	config *NetPollConfig
}

func NewEngine(handler ListenerEventHandler,config *NetPollConfig) *Engine {
	engine := &Engine{
		mds:    nil,
		config: config,
	}
}

