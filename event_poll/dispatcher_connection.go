package ddio

import (
	"errors"
	ch "github.com/zbh255/nyan/event_poll/internal/conn_handler"
	"reflect"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"
)

const (
	ONCE_MAX_EVENTS = 256
	BUFFER_SIZE = 4096
)

// ConnMultiEventDispatcher 从多路事件派发器
type ConnMultiEventDispatcher struct {
	handler ConnectionEventHandler
	poll EventLoop
	closed uint64
	done chan struct{}
	// 最多在事件循环中尝试读取的次数
	maxReadNumberOnEventLoop int
	// 最多在事件循环中尝试写入的次数
	maxWriteNumberOnEventLoop int
	// Block Number == 8192
	// Block Size == 4096 Byte
	/*
		从Reactor除了添加和删除事件之外和其他的goroutine
		之间并没有竞争，Http Server不涉及业务，应该不会将报文传递给其它goroutine
		所以，一些数据结构可以在栈本地直接分配，这样也是性能最高的。
		goroutine 栈最大可达1GB，所以一次为256个请求分配报文应该没有问题
	*/
	//memPool *MemoryPool
	//bufferPool *sync.Pool

}

// 对复用[]byte的一些描述
type bufferElem struct {
	buf []byte
}

func (b *bufferElem) Reset()  {
	b.buf = b.buf[:0]
}

func NewConnMultiEventDispatcher(handler ConnectionEventHandler) (*ConnMultiEventDispatcher,error) {
	cmed := &ConnMultiEventDispatcher{}
	cmed.handler = handler
	poller,err := NewPoller()
	if err != nil {
		logger.ErrorFromErr(err)
		return nil,err
	}
	cmed.maxReadNumberOnEventLoop = 1024
	cmed.maxWriteNumberOnEventLoop = 1024
	cmed.poll = poller
	// memory pool
	//cmed.memPool = NewBufferPool(12, int(math.Log2(ONCE_MAX_EVENTS)))
	//// buffer pool
	//cmed.bufferPool = &sync.Pool{
	//	New: func() interface{} {
	//		return &bufferElem{
	//			buf: make([]byte,BUFFER_SIZE),
	//		}
	//	},
	//}
	// open event loop
	go cmed.openLoop()
	return cmed,nil
}

func (p *ConnMultiEventDispatcher) AddConnEvent(ev *Event) error {
	err := p.poll.With(*ev)
	if err != nil {
		return err
	}
	return nil
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
	defer func() {
		p.done <- struct{}{}
	}()
	// 记录的待写入的Conn
	// 使用TCPConn而不使用*TCPConn的原因是防止对象逃逸
	writeConns := make(map[int]TCPConn,ONCE_MAX_EVENTS)
	for {
		// 检测关闭信号
		if atomic.LoadUint64(&p.closed) == 1 {
			return
		}
		events, err := p.poll.Exec(ONCE_MAX_EVENTS,time.Duration((time.Second * 2).Milliseconds()))
		//events, err := p.poll.Exec(ONCE_MAX_EVENTS,-1)
		if len(events) == 0 {
			continue
		}
		if err != syscall.EAGAIN && err != nil {
			break
		}
		// TODO: 暂时没有找到处理慢连接的好方法
		for _,v := range events {
			bc := &ch.BeforeConnHandler{}
			switch {
			case v.Flags() & EVENT_READ == EVENT_READ:
				var bufferTmp [BUFFER_SIZE]byte
				var buffer []byte
				buffer = *(*[]byte)(noescape(unsafe.Pointer(&reflect.SliceHeader{
					Data: uintptr(noescape(unsafe.Pointer(&bufferTmp))),
					Len:  BUFFER_SIZE,
					Cap:  BUFFER_SIZE,
				})))
				//buffer = buffer[:cap(buffer)]
				var tcpConn *TCPConn
				tcpConn = (*TCPConn)(noescape(unsafe.Pointer(&TCPConn{
					rawFd: int(v.fd()),
				})))
				bufferReadN := 0
				for i := 0; i < p.maxReadNumberOnEventLoop;i++ {
					readN, err := bc.NioRead(tcpConn.rawFd, buffer[bufferReadN:])
					bufferReadN += readN
					if err == syscall.EAGAIN || err == nil{
						tcpConn.rBytes = buffer
						tcpConn.rBytes = tcpConn.rBytes[:bufferReadN]
						// 分配写缓冲区
						var wBufferTmp [BUFFER_SIZE] byte
						var wBuffer []byte
						wBuffer = *(*[]byte)(noescape(unsafe.Pointer(&reflect.SliceHeader{
							Data: uintptr(noescape(unsafe.Pointer(&wBufferTmp))),
							Len:  BUFFER_SIZE,
							Cap:  BUFFER_SIZE,
						})))
						tcpConn.wBytes = wBuffer[:0]
						err := p.handler.OnData(tcpConn)
						if err != nil {
							p.handler.OnError(v,errors.New("OnData error: " + err.Error()))
							break
						}
						// 释放读缓冲
						tcpConn.rBytes = nil
						// 注册写事件
						p.modWrite(v)
						writeConns[tcpConn.rawFd] = *tcpConn
						break
					} else if err == syscall.EINTR {
						// 检查缓存区大小，容量满则扩容
						if !(len(buffer) == cap(buffer)) {
							continue
						}
						buffer := append(buffer,[]byte{0,0,0,0,0}...)
						buffer = buffer[:cap(buffer)]
						continue
					} else if err != nil {
						p.handler.OnError(v,ErrRead)
						break
					}
				}
			case v.Flags() & EVENT_WRITE == EVENT_WRITE:
				tcpConn,ok := writeConns[int(v.fd())]
				if !ok {
					logger.ErrorFromErr(errors.New("write event not register"))
					continue
				}
				for i := 0; i < p.maxWriteNumberOnEventLoop;i++ {
					writeN, err := bc.NioWrite(tcpConn.rawFd, tcpConn.wBytes)
					tcpConn.wBytes = tcpConn.wBytes[writeN:]
					if err != nil && err != syscall.EAGAIN {
						logger.ErrorFromErr(err)
						p.handler.OnError(v,ErrWrite)
						break
					}
					// 写完
					if len(tcpConn.wBytes) == 0 {
						break
					}
				}
				// 重新注册读事件
				p.modRead(v)
				// 不管出不出错都释放写缓冲区和记录写map key
				tcpConn.wBytes = nil
				delete(writeConns,tcpConn.rawFd)
			case v.Flags() & EVENT_CLOSE == EVENT_CLOSE:
				logger.Debug("client closed")
				_ = syscall.Close(int(v.fd()))
				break
			case v.Flags() & EVENT_ERROR == EVENT_ERROR:
				logger.Debug("connection error")
				_ = syscall.Close(int(v.fd()))
				break
			}
		}
	}
}

func (p *ConnMultiEventDispatcher) modRead(ev Event) {
	err := p.poll.Modify(Event{
		sysFd: ev.fd(),
		event: EVENT_READ | EVENT_CLOSE | EVENT_ERROR,
	})
	if err != nil {
		p.handler.OnError(Event{
			sysFd: ev.fd(),
			event: EVENT_READ | EVENT_CLOSE | EVENT_ERROR,
		},err)
	}
}

func (p *ConnMultiEventDispatcher) modWrite(ev Event) {
	err := p.poll.Modify(Event{
		sysFd: ev.fd(),
		event: EVENT_WRITE | EVENT_CLOSE | EVENT_ERROR,
	})
	if err != nil {
		p.handler.OnError(Event{
			sysFd: ev.fd(),
			event: EVENT_WRITE | EVENT_CLOSE | EVENT_ERROR,
		},err)
	}
}