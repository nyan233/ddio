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
	// 完成通知
	done chan struct{}
	// 一些主多路事件派发器的配置
	config *DisPatcherConfig
}

func NewListenerMultiEventDispatcher(handler ListenerEventHandler,onErr ErrorHandler,config *DisPatcherConfig) (*ListenerMultiEventDispatcher,error) {
	lmed := &ListenerMultiEventDispatcher{}
	// 启动绑定的从多路事件派发器
	connMds := make([]*ConnMultiEventDispatcher,runtime.NumCPU())
	for i := 0; i < len(connMds);i++ {
		connMds[i] = NewConnMultiEventDispatcher(config.ConnHandler,config.ConnErrHandler)
	}
	lmed.handler = handler
	lmed.config = config
	lmed.onErr = onErr
	lmed.poll = NewPoller(onErr)
	initEvent, err := lmed.handler.OnInit(config.EngineConfig)
	if err != nil {
		return nil,err
	}
	err = lmed.poll.With(initEvent)
	if err != nil {
		return nil, err
	}
	go lmed.openLoop()
	return lmed,nil
}

func (l *ListenerMultiEventDispatcher) Close() error {
	if !atomic.CompareAndSwapUint64(&l.closed,0,1) {
		return ErrorEpollClosed
	}
	<-l.done
	// 触发主多路事件派发器的定义的错误回调函数
	// 因为负责监听连接的Fd只有一个，所以直接取就好
	l.handler.OnError(l.poll.AllEvents()[0],ErrorEpollClosed)
	// 关闭所有子事件派发器
	for _,v := range l.connMds{
		v.Close()
	}
	return l.poll.Exit()
}

func (l *ListenerMultiEventDispatcher) openLoop() {
	for {
		if atomic.LoadUint64(&l.closed) == 1 {
			l.done <- struct{}{}
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

