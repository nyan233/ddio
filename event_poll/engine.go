package ddio

import (
	"context"
	"errors"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

// Engine 实例
type Engine struct {
	// 引擎绑定的多个监听多路事件派发器
	mds []*ListenerMultiEventDispatcher
	// 配置参数
	config *NetPollConfig
	// Main-Reactor & Sub-Reactor的Context取消函数
	cancelFn context.CancelFunc
	// Main-Reactor & Sub-Reactor的关闭计数
	wg sync.WaitGroup
}

func NewEngine(handler ListenerEventHandler, config *EngineConfig) (*Engine, error) {
	engine := &Engine{}
	nMds := runtime.NumCPU()
	if nMds > MAX_MASTER_LOOP_SIZE {
		nMds = MAX_MASTER_LOOP_SIZE
	}
	engine.mds = make([]*ListenerMultiEventDispatcher, nMds)
	// 程序是否监听了多个端口
	var isMultiAddr = len(config.MultiAddr) > 1
	netPollConfigs := make([]NetPollConfig, 0, len(config.MultiAddr))
	argMaps := make(map[int]map[string]string, len(config.MultiAddr))
	for k, v := range config.MultiAddr {
		netPollConfig, argMap, err := parseAddress(v)
		argMaps[k] = argMap
		netPollConfig.IsMultiAddr = isMultiAddr
		if err != nil {
			return nil, err
		}
		netPollConfigs = append(netPollConfigs, netPollConfig)
	}
	// 根据设置的Level来分配监听线程的数量
	addrNMds := make(map[int]int, len(config.MultiAddr))
	var newNMds int
	for k, v := range argMaps {
		for k2, v2 := range v {
			if strings.EqualFold(k2, "level") {
				level, err := strconv.Atoi(v2)
				if err != nil {
					return nil, errors.New("level is bad value")
				}
				level = nMds * level / 10
				if level == 0 {
					level = 1
				}
				addrNMds[k] = level
				newNMds += level
			}
		}
	}
	// 根据新确定的监听线程数量调整mds
	engine.mds = make([]*ListenerMultiEventDispatcher, 0, newNMds)
	// context
	ctx,cancel := context.WithCancel(context.Background())
	engine.cancelFn = cancel
	// WaitGroup
	engine.wg = sync.WaitGroup{}
	engine.wg.Add(newNMds)
	// 创建对应数量的监听线程
	for k, v := range addrNMds {
		for i := 0; i < v; i++ {
			tmp, err := NewListenerMultiEventDispatcher(ctx, &engine.wg, handler, &ListenerConfig{
				ConnEHd:       config.ConnHandler,
				Balance:       config.NBalance(),
				NetPollConfig: &netPollConfigs[k],
			})
			if err != nil {
				return nil, err
			}
			engine.mds = append(engine.mds, tmp)
		}
	}

	return engine, nil
}

func (e *Engine) Close() error {
	e.cancelFn()
	e.wg.Wait()
	return nil
}
