package ddio

import (
	"context"
	"runtime"
	"sync"
	"syscall"
)

// ListenerMultiEventDispatcher 主多路事件派发器
type ListenerMultiEventDispatcher struct {
	// 提升关闭的速度
	wg *sync.WaitGroup
	// 子Reactor的Count
	subWg sync.WaitGroup
	// context用于通知关闭
	// 该Context也会一起被派生到子Reactor中
	// 所以上层调用CancelFunc时，子Reactor也会感受到关闭事件
	ctx context.Context
	handler ListenerEventHandler
	poll    EventLoop
	// 与监听事件多路事件派发器绑定的连接多路事件派发器
	connMds []*ConnMultiEventDispatcher
	// 一些主多路事件派发器的配置
	config *ListenerConfig
}

func NewListenerMultiEventDispatcher(ctx context.Context, wg *sync.WaitGroup, handler ListenerEventHandler, config *ListenerConfig) (*ListenerMultiEventDispatcher, error) {
	lmed := &ListenerMultiEventDispatcher{}
	// 启动绑定的从多路事件派发器
	nMds := runtime.NumCPU()
	if nMds > MAX_SLAVE_LOOP_SIZE {
		nMds = MAX_SLAVE_LOOP_SIZE
	}
	// 所有子Goroutine共享的Pool
	pool := sync.Pool{
		New: func() interface{} {
			return make([]byte, BUFFER_SIZE)
		},
	}
	connMds := make([]*ConnMultiEventDispatcher, nMds)
	connConfig := config.ConnEHd.OnInit()
	// Sub-Reactor WaitGroup
	lmed.subWg = sync.WaitGroup{}
	lmed.subWg.Add(nMds)
	for i := 0; i < len(connMds); i++ {
		subCtx := context.WithValue(ctx,0,0)
		tmp, err := NewConnMultiEventDispatcher(subCtx, &lmed.subWg, config.ConnEHd, connConfig)
		tmp.bufferPool = &pool
		if err != nil {
			return nil, err
		}
		connMds[i] = tmp
	}
	lmed.wg = wg
	lmed.ctx = ctx
	lmed.connMds = connMds
	lmed.handler = handler
	lmed.config = config
	poller, err := NewPoller()
	if err != nil {
		logger.ErrorFromErr(err)
		return nil, err
	}
	lmed.poll = poller
	initEvent, err := lmed.handler.OnInit(config.NetPollConfig)
	if err != nil {
		return nil, err
	}
	err = lmed.poll.With(*initEvent)
	if err != nil {
		return nil, err
	}
	go lmed.openLoop()
	return lmed, nil
}


func (l *ListenerMultiEventDispatcher) openLoop() {
	defer func() {
		l.wg.Done()
	}()
	receiver := make([]Event, 1)
	for {
		// 在事件循环里检测关闭
		select {
		case <-l.ctx.Done():
			// 触发主多路事件派发器的定义的错误回调函数
			// 因为负责监听连接的Fd只有一个，所以直接取就好
			l.handler.OnError(l.poll.AllEvents()[0], ErrorEpollClosed)
			// 等待绑定的子Reactor关闭
			l.subWg.Wait()
			// 关闭Poller
			// TODO 想办法传递Poller关闭时的错误
			_ = l.poll.Exit()
			return
		default:
			break
		}
		events, err := l.poll.Exec(receiver, EVENT_LOOP_SLEEP)
		if events == 0 {
			continue
		}
		if err != syscall.EAGAIN && err != nil {
			break
		}
		event := receiver[0]
		connFd, err := l.handler.OnAccept(event)
		if err != nil {
			logger.ErrorFromString("accept error: " + err.Error())
			continue
		}
		connEvent := &Event{
			sysFd: int32(connFd),
			event: EVENT_READ | EVENT_CLOSE | EVENT_ERROR,
		}
		n := l.config.Balance.Target(len(l.connMds), connFd)
		err = l.connMds[n].AddConnEvent(connEvent)
		if err != nil {
			logger.ErrorFromString("add connection event error : " + err.Error())
		}
	}
}
