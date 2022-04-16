package ddio

import "runtime"

// Engine 实例
type Engine struct {
	// 引擎绑定的多个监听多路事件派发器
	mds []*ListenerMultiEventDispatcher
	// 配置参数
	config *NetPollConfig
}

func NewEngine(handler ListenerEventHandler,config *DisPatcherConfig) (*Engine,error) {
	engine := &Engine{}
	engine.mds = make([]*ListenerMultiEventDispatcher,runtime.NumCPU())
	for k := range engine.mds {
		tmp, err := NewListenerMultiEventDispatcher(handler,config)
		if err != nil {
			return nil,err
		}
		engine.mds[k] = tmp
	}
	engine.config = config.EngineConfig
	return engine,nil
}

func (e *Engine) Close() error {
	for _,v := range e.mds {
		err := v.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func (* Engine) Run() {
	select {}
}
