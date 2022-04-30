package ddio

import (
	"errors"
	ch "github.com/zbh255/nyan/event_poll/internal/conn_handler"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

const (
	ONCE_MAX_EVENTS = 1024
	BUFFER_SIZE     = 4096
)

// ConnMultiEventDispatcher 从多路事件派发器
type ConnMultiEventDispatcher struct {
	handler ConnectionEventHandler
	poll    EventLoop
	closed  uint64
	done    chan struct{}
	connConfig ConnConfig
	/*
		从Reactor除了添加和删除事件之外和其他的goroutine
		之间并没有竞争，Http Server不涉及业务，应该不会将报文传递给其它goroutine
		所以，一些数据结构可以在栈本地直接分配，这样也是性能最高的。
		goroutine 栈最大可达1GB，所以一次为256个请求分配报文应该没有问题
	*/
	// 为in-out准备的内存池，它主要用于准备默认大小的Buffer
	littleMemPool *MemoryPool
	// 为in-out准备的内存池，它可以优化Big-Http-Header之类的场景
	bigMemPool *MemoryPool
	// 所有子Reactor共享的Pool
	// 该值应该由主Reactor初始化时设置
	bufferPool *sync.Pool

}

// 对复用[]byte的一些描述
type bufferElem struct {
	buf []byte
}

func (b *bufferElem) Reset() {
	b.buf = b.buf[:0]
}

func NewConnMultiEventDispatcher(handler ConnectionEventHandler,connConfig ConnConfig) (*ConnMultiEventDispatcher, error) {
	cmed := &ConnMultiEventDispatcher{}
	cmed.handler = handler
	poller, err := NewPoller()
	if err != nil {
		logger.ErrorFromErr(err)
		return nil, err
	}
	cmed.connConfig = connConfig
	cmed.poll = poller
	// memory pool
	//cmed.bigMemPool = NewBufferPool(12, int(math.Log2(ONCE_MAX_EVENTS)))
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
	return cmed, nil
}

func (p *ConnMultiEventDispatcher) AddConnEvent(ev *Event) error {
	err := p.poll.With(*ev)
	if err != nil {
		return err
	}
	return nil
}

func (p *ConnMultiEventDispatcher) Close() {
	if !atomic.CompareAndSwapUint64(&p.closed, 0, 1) {
		// 不允许重复关闭
		return
	}
	<-p.done
	for _, v := range p.poll.AllEvents() {
		p.handler.OnError(v, ErrorEpollClosed)
	}
}

func (p *ConnMultiEventDispatcher) openLoop() {
	defer func() {
		p.done <- struct{}{}
	}()
	// 记录的待写入的Conn
	// 使用TCPConn而不使用*TCPConn的原因是防止对象逃逸
	writeConns := make(map[int]TCPConn, ONCE_MAX_EVENTS)
	freeWConn := func(fd int) {
		delete(writeConns,fd)
	}
	// 堆分配的in-out buffer，大小是默认栈分配的两倍，即8192
	// 分割为1024块缓存区
	p.bigMemPool = NewBufferPool(13,10)
	// 小内存池，大小为4096
	p.littleMemPool = NewBufferPool(12,10)
	receiver := make([]Event, ONCE_MAX_EVENTS)
	for {
		// 检测关闭信号
		if atomic.LoadUint64(&p.closed) == 1 {
			return
		}
		nEvent, err := p.poll.Exec(receiver, time.Second * 2)
		//events, err := p.poll.Exec(ONCE_MAX_EVENTS,-1)
		if nEvent == 0 {
			continue
		}
		if err != syscall.EAGAIN && err != nil {
			break
		}
		events := receiver[:nEvent]
		// TODO: 暂时没有找到处理慢连接的好方法
		for _, v := range events {
			switch {
			case v.Flags()&EVENT_CLOSE == EVENT_CLOSE:
				logger.Debug("client closed")
				_ = syscall.Close(int(v.fd()))
				freeWConn(int(v.fd()))
				break
			case v.Flags()&EVENT_ERROR == EVENT_ERROR:
				logger.Debug("connection error")
				_ = syscall.Close(int(v.fd()))
				freeWConn(int(v.fd()))
				break
			case v.Flags()&EVENT_READ == EVENT_READ:
				p.handlerReadEvent(v,writeConns)
			case v.Flags()&EVENT_WRITE == EVENT_WRITE:
				p.handlerWriteEvent(v,writeConns)
			}
		}
	}
}

// wPoolAlloc指示写缓冲区是否从p.bufferPool分配，该成员类型是*sync.Pool
func (p *ConnMultiEventDispatcher) handlerReadEvent(ev Event,writeConns map[int]TCPConn) (wPoolAlloc bool){
	bc := ch.BeforeConnHandler{}
	buffer,ok := p.littleMemPool.AllocBuffer(1)
	var rPoolAlloc bool
	if !ok {
		buffer = p.bufferPool.Get().([]byte)[:0]
		rPoolAlloc = true
	}
	//buffer = buffer[:cap(buffer)]
	var tcpConn *TCPConn
	tcpConn = &TCPConn{
		rawFd: int(ev.fd()),
		appendFn: p.appendBytes,
		freeFn: p.freeBytes,
	}
	bufferReadN := 0
	var onDataOk bool
	readEvent:
	for i := 0; i < p.connConfig.MaxReadSysCallNumberOnEventLoop; i++ {
		readN, err := bc.NioRead(tcpConn.rawFd, buffer[bufferReadN:])
		bufferReadN += readN
		if onDataOk {
			err = syscall.EAGAIN
		}
		if err == syscall.EAGAIN || err == nil {
			tcpConn.rBytes = buffer
			tcpConn.rBytes = tcpConn.rBytes[:bufferReadN]
			// 分配写缓冲区
			wBuffer,bl := p.littleMemPool.AllocBuffer(1)
			if !bl {
				wBuffer = p.bufferPool.Get().([]byte)
				wPoolAlloc = true
			}
			tcpConn.wBytes = wBuffer[:0]
			err := p.handler.OnData(tcpConn)
			if err != nil {
				p.handler.OnError(ev, errors.New("OnData error: "+err.Error()))
				break
			}
			// 释放读缓冲
			if rPoolAlloc {
				p.bufferPool.Put(buffer)
			} else {
				if p.littleMemPool.IsAlloc(buffer) {
					p.littleMemPool.FreeBuffer(&buffer)
				} else if p.bigMemPool.IsAlloc(buffer) {
					p.bigMemPool.FreeBuffer(&buffer)
				}
			}
			tcpConn.rBytes = nil
			// 写缓冲区有数据时则注册写事件
			if len(tcpConn.wBytes) > 0 {
				p.modWrite(ev)
				writeConns[tcpConn.rawFd] = *tcpConn
			}
			break
		} else if err == syscall.EINTR {
			// 检查缓存区大小，容量满则扩容
			if !(len(buffer) == cap(buffer)) {
				continue
			}
			// 检查是否符合触发OnData事件需要读取的Buffer-Block数量
			if len(buffer) / BUFFER_SIZE >= p.connConfig.OnDataNBlock {
				goto readEvent
			}
			var growOk bool
			// 针对小缓存区的扩容操作
			// 分配大缓存区的空间->分配失败则从bufferPool分配->释放原有空间和标记分配情况
			if p.littleMemPool.IsAlloc(buffer) {
				newBuf,bl := p.bigMemPool.AllocBuffer(1)
				growOk = bl
				if !bl {
					newBuf = p.bufferPool.Get().([]byte)[:0]
					newBuf = append(newBuf,buffer...)
					rPoolAlloc = true
					p.littleMemPool.FreeBuffer(&buffer)
					buffer = newBuf
					growOk = true
				}
			}
			// 如果不判断是否已经扩容的话，就会导致重复扩容
			if !growOk && p.bigMemPool.IsAlloc(buffer) {
				newBuf,bl := doubleGrow(p.bigMemPool,buffer)
				if !bl {
					newBuf = p.bufferPool.Get().([]byte)[:0]
					newBuf = append(newBuf,buffer...)
					rPoolAlloc = true
					p.bigMemPool.FreeBuffer(&buffer)
					buffer = newBuf
				}
			}
			continue
		} else if err != nil {
			p.handler.OnError(ev, ErrRead)
			break
		}
	}
	return
}

func (p *ConnMultiEventDispatcher) handlerWriteEvent(ev Event,writeConns map[int]TCPConn) {
	bc := &ch.BeforeConnHandler{}
	tcpConn, ok := writeConns[int(ev.fd())]
	if !ok {
		logger.ErrorFromErr(errors.New("write event not register"))
		return
	}
	wb := tcpConn.wBytes
	for i := 0; i < p.connConfig.MaxWriteSysCallNumberOnEventLoop; i++ {
		writeN, err := bc.NioWrite(tcpConn.rawFd, wb)
		wb = wb[writeN:]
		if err != nil && err != syscall.EAGAIN {
			logger.ErrorFromErr(err)
			p.handler.OnError(ev, ErrWrite)
			break
		}
		// 写完
		if len(wb) == 0 {
			break
		}
	}
	// 重新注册读事件
	p.modRead(ev)
	// 不管出不出错都释放写缓冲区和记录写map key
	if p.littleMemPool.IsAlloc(tcpConn.wBytes) {
		p.littleMemPool.FreeBuffer(&tcpConn.wBytes)
	} else if p.bigMemPool.IsAlloc(tcpConn.wBytes) {
		p.bigMemPool.FreeBuffer(&tcpConn.wBytes)
	} else {
		p.bufferPool.Put(tcpConn.wBytes)
	}
	tcpConn.wBytes = nil
	delete(writeConns, tcpConn.rawFd)
}

func (p *ConnMultiEventDispatcher) appendBytes(oldBuf []byte) (newBuf []byte,bl bool) {
	if p.littleMemPool.IsAlloc(oldBuf) {
		newBuf,bl = doubleGrow(p.littleMemPool,oldBuf)
	} else if p.bigMemPool.IsAlloc(oldBuf) {
		newBuf,bl = doubleGrow(p.bigMemPool,oldBuf)
	}
	return
}

func (p *ConnMultiEventDispatcher) freeBytes(oldBuf []byte) {
	if p.littleMemPool.IsAlloc(oldBuf) {
		p.littleMemPool.FreeBuffer(&oldBuf)
	} else if p.bigMemPool.IsAlloc(oldBuf) {
		p.bigMemPool.FreeBuffer(&oldBuf)
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
		}, err)
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
		}, err)
	}
}
