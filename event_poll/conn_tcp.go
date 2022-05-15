package ddio

import (
	"errors"
	"net"
	"sync/atomic"
	"time"
)

var (
	ErrConnClosed = errors.New("conn is closed")
)

type TCPConn struct {
	// 文件描述符
	// fd必须确保没有被dup之类的系统调用复制
	rawFd int
	// 读缓冲
	rBytes []byte
	// 写缓冲
	wBytes     []byte
	hd         AfterHandler
	nextNBlock int
	// 告诉事件循环，用户代码已经将连接关闭
	closed uint32
	//bigMemPool *MemoryPool
	//bufferPool *sync.Pool
	// 注册的专门用于扩容缓存区的函数
	appendFn func(oldBuf []byte) (newBuf []byte, bl bool)
	// 用于释放缓存区空间的函数
	freeFn func(buf []byte)
	// sync.Pool分配的缓冲元素
	//buf *bufferElem
	addr net.Addr
	// 定时器，用于设置连接死线相关功能
	timer *ddTimer
}

func (T *TCPConn) TakeReadBytes() []byte {
	return T.rBytes
}

func (T *TCPConn) TakeWriteBuffer() *[]byte {
	if cap(T.wBytes) == 0 {
		return nil
	} else {
		return &T.wBytes
	}
}

func (T *TCPConn) WriteBytes(p []byte) {
	for len(p)+len(T.wBytes) > cap(T.wBytes) {
		newBuf, bl := T.appendFn(T.wBytes)
		if bl {
			T.wBytes = newBuf
		} else {
			oldBuf := T.wBytes
			T.wBytes = make([]byte, 0, cap(oldBuf)*2)
			T.wBytes = append(T.wBytes, oldBuf...)
			T.freeFn(oldBuf)
		}
	}
	T.wBytes = append(T.wBytes, p...)
}

func (T *TCPConn) RegisterAfterHandler(hd AfterHandler) {
	T.hd = hd
}

func (T *TCPConn) Next(nBlock int) {
	T.nextNBlock = nBlock
}

// Close 设置一个关闭标志，事件循环会审查这个标志，在写入完缓存区的数据或者出错时会将其关闭
func (T *TCPConn) Close() error {
	if !atomic.CompareAndSwapUint32(&T.closed,0,1) {
		return ErrConnClosed
	}
	return nil
}

func (T *TCPConn) Addr() net.Addr {
	return T.addr
}

func (T *TCPConn) timeoutHandler(data interface{},timeOut time.Duration) {
	_ = T.Close()
}

func (T *TCPConn) SetDeadLine(deadline time.Duration) error {
	return T.timer.AddTimer(true,deadline,0,T.timeoutHandler)
}

func (T *TCPConn) SetTimeout(timeout time.Duration) error {
	return T.timer.AddTimer(false,timeout,0,T.timeoutHandler)
}
