package ddio

import (
	"runtime"
	"sync/atomic"
	"time"
)

// ListenerMultiEventDispatcher 主多路事件派发器
type ListenerMultiEventDispatcher struct {
	handler ListenerEventHandler
	poll EventLoop
	// 错误处理函数
	onErr ErrorHandler
	// 与监听事件多路事件派发器绑定的连接多路事件派发器
	connMds []*ConnMultiEventDispatcher
	// 关闭标志
	closed uint64
	// 一些主多路事件派发器的配置
	config *DisPatcherConfig
}

func NewListenerMultiEventDispatcher(handler ListenerEventHandler,config *DisPatcherConfig) *ListenerMultiEventDispatcher {
	lmed := &ListenerMultiEventDispatcher{}
	// 启动绑定的从多路事件派发器
	connMds := make([]*ConnMultiEventDispatcher,runtime.NumCPU())
	for i := 0; i < len(connMds);i++ {
		connMds[i] = NewConnMultiEventDispatcher(config.ConnHandler,config.ConnErrHandler)
	}
	lmed.handler = handler
	lmed.config = config
	go lmed.openLoop()
	return lmed
}

func (l *ListenerMultiEventDispatcher) openLoop() {
	for {
		if atomic.LoadUint64(&l.closed) == 1 {
			return
		}
		events, err := l.poll.Exec(1, time.Second*2)
		if err != nil {
			l.onErr(err)
		}
		event := events[0]
		connFd, err := l.handler.OnAccept(event)
		if err != nil {
			l.onErr(err)
		}
		connEvent := &Event{
			sysFd: int32(connFd),
			event: l.config.ConnEvent,
		}
		err = l.connMds[connFd%len(l.connMds)].AddConnEvent(connEvent)
		if err != nil {
			l.onErr(err)
		}
	}
}

