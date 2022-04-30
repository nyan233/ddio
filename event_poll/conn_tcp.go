package ddio

import (
	"net"
	"syscall"
	"time"
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
	closed bool
	//bigMemPool *MemoryPool
	//bufferPool *sync.Pool
	// 注册的专门用于扩容缓存区的函数
	appendFn func(oldBuf []byte) (newBuf []byte, bl bool)
	// 用于释放缓存区空间的函数
	freeFn func(buf []byte)
	// sync.Pool分配的缓冲元素
	//buf *bufferElem
	addr net.Addr
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

func (T *TCPConn) Close() error {
	T.closed = true
	return syscall.Close(T.rawFd)
}

func (T *TCPConn) Addr() net.Addr {
	return T.addr
}

func (T *TCPConn) SetDeadLine(deadline time.Time) error {
	panic("implement me")
}

func (T *TCPConn) SetTimeout(timeout time.Duration) error {
	panic("implement me")
}
