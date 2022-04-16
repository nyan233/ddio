package ddio

import (
	ch "github.com/zbh255/nyan/event_poll/internal/conn_handler"
	"sync/atomic"
	"syscall"
	"time"
)

const (
	ONCE_MAX_EVENTS = 256
	BUFFER_SIZE = 512
)

// ConnMultiEventDispatcher 从多路事件派发器
type ConnMultiEventDispatcher struct {
	handler ConnectionEventHandler
	poll EventLoop
	closed uint64
	done chan struct{}
	// Block Number == 8192
	// Block Size == 4096 Byte
	bufferPool *BufferPool
}

func NewConnMultiEventDispatcher(handler ConnectionEventHandler) (*ConnMultiEventDispatcher,error) {
	cmed := &ConnMultiEventDispatcher{}
	cmed.handler = handler
	poller,err := NewPoller()
	if err != nil {
		logger.ErrorFromErr(err)
		return nil,err
	}
	cmed.poll = poller
	// buffer pool
	cmed.bufferPool = NewBufferPool(12,13)
	// open event loop
	go cmed.openLoop()
	return cmed,nil
}

func (p *ConnMultiEventDispatcher) AddConnEvent(ev *Event) error {
	err := p.poll.With(ev)
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
	writeConns := make(map[int]*TCPConn,ONCE_MAX_EVENTS)
	for {
		// 检测关闭信号
		if atomic.LoadUint64(&p.closed) == 1 {
			return
		}
		events, err := p.poll.Exec(ONCE_MAX_EVENTS,time.Duration((time.Second * 2).Milliseconds()))
		if len(events) == 0 {
			continue
		}
		if err != syscall.EAGAIN && err != nil {
			break
		}
		// TODO: 暂时没有找到处理慢连接的好方法
		// 没有读取完数据的套接字map
		readyRSockets := make(map[int]*TCPConn,len(events))
		// 此次被触发的写就绪的Socket
		readyWSockets := make(map[int]*TCPConn,len(events))
		// 遍历所有就绪的Events
		// 由于Epoll使用的是ET的触发模式，所以这是初次处理
		for _,v := range events {
			// 检查触发的事件类型
			// 触发写事件则写入数据
			if v.Flags() & EVENT_WRITE == EVENT_WRITE {
				conn,ok := writeConns[int(v.fd())]
				if !ok {
					logger.PanicFromString("register write event is not save")
				}
				bc := &ch.BeforeConnHandler{}
				writeN, err := bc.NioWrite(conn.rawFd, conn.wBytes)
				if err != syscall.EAGAIN && err != nil {
					logger.ErrorFromErr(err)
					continue
				}
				conn.wBytes = conn.wBytes[writeN:]
				// 检查有无写完写缓冲区
				if len(conn.wBytes) == 0 {
					p.bufferPool.FreeBuffer(&conn.wBytes)
					conn.wBytes = nil
					delete(writeConns,conn.rawFd)
					// 写完则重新注册读事件
					err = p.poll.Modify(&Event{
						sysFd: v.fd(),
						event: EVENT_READ,
					})
					if err != nil {
						logger.ErrorFromErr(err)
						return
					}
				} else {
					readyWSockets[conn.rawFd] = conn
				}
				continue
			}
			// 池没有空余空间的时候则重新分配
			buffer,ok := p.bufferPool.AllocBuffer(1)
			if !ok {
				buffer = make([]byte,BUFFER_SIZE)
			}
			buffer = buffer[:cap(buffer)]
			bc := &ch.BeforeConnHandler{}
			readN, err := bc.NioRead(int(v.fd()), buffer)
			switch {
			case err == syscall.EAGAIN:
				readyRSockets[int(v.fd())] = &TCPConn{
					rawFd:   int(v.fd()),
					rBytes:  buffer[:readN],
					wBytes:  nil,
					memPool: p.bufferPool,
				}
				break
			case err == nil:
				wBytes,ok := p.bufferPool.AllocBuffer(1)
				if !ok {
					wBytes = make([]byte,BUFFER_SIZE)
				}
				tcpConn := &TCPConn{
					rawFd: int(v.fd()),
					rBytes:  buffer[:readN],
					wBytes:  wBytes,
					memPool: p.bufferPool,
				}
				err := p.handler.OnData(tcpConn)
				if err != nil {
					logger.ErrorFromErr(err)
					logger.ErrorFromErr(p.handler.OnClose(v))
				}
				// 释放读缓冲区
				p.bufferPool.FreeBuffer(&buffer)
				// 将注册的读事件修改为写事件
				err = p.poll.Modify(&Event{
					sysFd: v.fd(),
					event: EVENT_WRITE,
				})
				if err != nil {
					return
				}
				writeConns[tcpConn.rawFd] = tcpConn
			default:
				p.handler.OnError(v,err)
			}
		}
		// 遍历处理所有未读取完的Socket
		for len(readyRSockets) != 0 && len(readyWSockets) != 0 {
			for _,conn := range readyWSockets {
				bc := &ch.BeforeConnHandler{}
				writeN, err := bc.NioWrite(conn.rawFd, conn.wBytes)
				if err != syscall.EAGAIN && err != nil {
					logger.ErrorFromErr(err)
					continue
				}
				conn.wBytes = conn.wBytes[writeN:]
				// 检查有无写完写缓冲区
				if len(conn.wBytes) == 0 {
					p.bufferPool.FreeBuffer(&conn.wBytes)
					conn.wBytes = nil
					delete(writeConns,conn.rawFd)
					delete(readyWSockets,conn.rawFd)
					// 写完则重新注册读事件
					err := p.poll.Modify(&Event{
						sysFd: int32(conn.rawFd),
						event: EVENT_READ,
					})
					if err != nil {
						logger.ErrorFromErr(err)
						return
					}
				}
			}
			for k,v := range readyRSockets {
				// 读缓冲区满则扩容
				if len(v.rBytes) == cap(v.rBytes) {
					ok := p.bufferPool.Grow(&v.rBytes,(cap(v.rBytes)/int(p.bufferPool.block)) * 2)
					// 缓冲区没有空余的内存则需重新分配
					if !ok {
						oldLen := len(v.rBytes)
						v.rBytes = append(v.rBytes,[]byte{0,0,0,0,0}...)
						v.rBytes = v.rBytes[:oldLen]
					}
				}
				bc := &ch.BeforeConnHandler{}
				readN, err := bc.NioRead(k, v.rBytes[len(v.rBytes):])
				switch {
				case err == nil:
					// 分配写缓冲区
					wBytes,ok := p.bufferPool.AllocBuffer(1)
					if !ok {
						wBytes = make([]byte,BUFFER_SIZE)
					}
					v.wBytes = wBytes
					if err := p.handler.OnData(v);err != nil {
						logger.ErrorFromErr(err)
						p.handler.OnError(Event{
							sysFd: int32(k),
							event: 0,
						},err)
					}
					// 释放读缓冲区
					p.bufferPool.FreeBuffer(&v.rBytes)
					v.rBytes = nil
					// 删除记录的等待读取的Conn
					delete(readyRSockets,k)
					// 将注册的读事件修改为写事件
					err = p.poll.Modify(&Event{
						sysFd: int32(v.rawFd),
						event: EVENT_WRITE,
					})
					if err != nil {
						logger.ErrorFromErr(err)
						return
					}
					// 记录写事件
					writeConns[k] = v
				case err == syscall.EAGAIN:
					v.rBytes = v.rBytes[:len(v.rBytes) + readN]
					break
				default:
					// 释放读缓冲区
					p.bufferPool.FreeBuffer(&v.rBytes)
					// 触发用户注册的用于处理错误的方法
					p.handler.OnError(Event{
						sysFd: int32(k),
						event: 0,
					},err)
					// 在等待读取的Conn中删除此连接
					delete(readyRSockets,k)
					break
				}
			}
		}
		//// 处理所有待写入的Socket Conn
		//for len(readyWSockets) != 0 {
		//
		//}
	}
}
