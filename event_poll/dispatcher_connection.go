package ddio

import (
	"golang.org/x/sys/unix"
	"io"
	"sync/atomic"
	"time"
)

const (
	ONCE_MAX_EVENTS = 256
	BUFFER_SIZE = 512
)

// ConnMultiEventDispatcher 从多路事件派发器
type ConnMultiEventDispatcher struct {
	handler ConnectionEventHandler
	onErr ErrorHandler
	poll EventLoop
	closed uint64
	done chan struct{}
}

func NewConnMultiEventDispatcher(handler ConnectionEventHandler,onErr func(err error)) *ConnMultiEventDispatcher {
	cmed := &ConnMultiEventDispatcher{}
	cmed.handler = handler
	poller := NewPoller(onErr)
	cmed.poll = poller
	cmed.onErr = onErr
	// open event loop
	go cmed.openLoop()
	return cmed
}

func (p *ConnMultiEventDispatcher) AddConnEvent(ev *Event) error {
	return p.poll.With(ev)
}

func (p *ConnMultiEventDispatcher) Close() {
	if !atomic.CompareAndSwapUint64(&p.closed,0,1) {
		// 不允许重复关闭
		return
	}
	<-p.done
	for _,v := range p.poll.AllEvents() {
		p.handler.OnError(v,ErrorEpollClosed)
	}
}

func (p *ConnMultiEventDispatcher) openLoop() {
	for {
		// 检测关闭信号
		if atomic.LoadUint64(&p.closed) == 1 {
			p.done <- struct{}{}
			return
		}
		events, err := p.poll.Exec(ONCE_MAX_EVENTS,time.Second * 2)
		if err != nil {
			p.onErr(err)
			break
		}
		buffer := make([]byte,BUFFER_SIZE)
		for _,v := range events{
			for {
				readN := 0
				n, err := unix.Read(int(v.fd()),buffer)
				if err != nil {
					continue
				}
				for n == BUFFER_SIZE {
					readN += n
					buffer = append(buffer, make([]byte, BUFFER_SIZE)...)
					n, err = unix.Read(int(v.fd()),buffer[readN : readN+BUFFER_SIZE])
					if err == io.EOF {
						n = 0
						break
					}
					if err != nil {
						continue
					}
				}
				readN += n
				buffer = buffer[:readN]
				break
			}
			wb := make([]byte,512)
			if v.Flags() & EVENT_READ == EVENT_READ {
				err := p.handler.OnRead(buffer, wb, v.Flags())
				if err != nil {
					p.onErr(err)
				}
			}
		}
	}
}
