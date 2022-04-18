package ddio

import "runtime"

// Engine 实例
type Engine struct {
	// 引擎绑定的多个监听多路事件派发器
	mds []*ListenerMultiEventDispatcher
	// 配置参数
	config *NetPollConfig
}

func NewEngine(handler ListenerEventHandler,config *EngineConfig) (*Engine,error) {
	engine := &Engine{}
	nMds := runtime.NumCPU()
	if nMds > MAX_MASTER_LOOP_SIZE {
		nMds = MAX_MASTER_LOOP_SIZE
	}
	engine.mds = make([]*ListenerMultiEventDispatcher,nMds)
	for k := range engine.mds {
		tmp, err := NewListenerMultiEventDispatcher(handler,&ListenerConfig{
			ConnEHd:       config.ConnHandler,
			Balance:       config.NBalance(),
			NetPollConfig: config.NetPollConfig,
		})
		if err != nil {
			return nil,err
		}
		engine.mds[k] = tmp
	}
	engine.config = config.NetPollConfig
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

