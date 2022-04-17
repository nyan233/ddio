package ddio

import (
	"runtime"
	"sync/atomic"
	"syscall"
	"time"
)

// ListenerMultiEventDispatcher 主多路事件派发器
type ListenerMultiEventDispatcher struct {
	handler ListenerEventHandler
	poll EventLoop
	// 与监听事件多路事件派发器绑定的连接多路事件派发器
	connMds []*ConnMultiEventDispatcher
	// 关闭标志
	closed uint64
	// 完成通知
	done chan struct{}
	// 一些主多路事件派发器的配置
	config *ListenerConfig
}

func NewListenerMultiEventDispatcher(handler ListenerEventHandler,config *ListenerConfig) (*ListenerMultiEventDispatcher,error) {
	lmed := &ListenerMultiEventDispatcher{}
	// 启动绑定的从多路事件派发器
	connMds := make([]*ConnMultiEventDispatcher,runtime.NumCPU())
	for i := 0; i < len(connMds);i++ {
		tmp, err := NewConnMultiEventDispatcher(config.ConnEHd)
		if err != nil {
			return nil, err
		}
		connMds[i] = tmp
	}
	lmed.connMds = connMds
	lmed.handler = handler
	lmed.config = config
	poller,err := NewPoller()
	if err != nil {
		logger.ErrorFromErr(err)
		return nil,err
	}
	lmed.poll = poller
	initEvent, err := lmed.handler.OnInit(config.NetPollConfig)
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
		events, err := l.poll.Exec(1, time.Duration((time.Second * 2).Milliseconds()))
		if len(events) == 0 {
			continue
		}
		if err != syscall.EAGAIN && err != nil {
			break
		}
		event := events[0]
		connFd, err := l.handler.OnAccept(event)
		if err != nil {
			logger.ErrorFromString("accept error: " + err.Error())
			continue
		}
		connEvent := &Event{
			sysFd: int32(connFd),
			event: EVENT_READ | EVENT_CLOSE,
		}
		n := l.config.Balance.Target(len(l.connMds),connFd)
		err = l.connMds[n].AddConnEvent(connEvent)
		if err != nil {
			logger.ErrorFromString("add connection event error : " + err.Error())
			break
		}
	}
}

