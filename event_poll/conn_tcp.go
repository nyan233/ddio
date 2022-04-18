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
	hd         ConnAfterCenterHandler
	nextNBlock int
	// 告诉事件循环，用户代码已经将连接关闭
	closed bool
	//memPool *MemoryPool
	//bufferPool *sync.Pool
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

// GrowWriteBuffer 扩容大小 == 原来大小 + nCap * Block
func (T *TCPConn) GrowWriteBuffer(buf *[]byte, nCap int) {
	//if !T.memPool.Grow(buf,(cap(*buf) / T.memPool.BlockSize()) + nCap) {
	//	tmp := make([]byte,cap(*buf) * 2)
	//	tmp = tmp[:len(*buf)]
	//	copy(tmp,*buf)
	//	// 扩容失败可能是内存池的容量不足导致扩容失败
	//	// 这时，如果是由内存池分配的内存不释放就会导致内存泄漏
	//	// 所以，以下这段代码负责内存池分配内存的释放工作
	//	if T.memPool.IsAlloc(*buf) {
	//		T.memPool.FreeBuffer(buf)
	//	}
	//	*buf = tmp
	//	T.buf.buf = tmp
	//}

}

func (T *TCPConn) WriteBytes(p []byte) {
	T.wBytes = append(T.wBytes, p...)
}

func (T *TCPConn) RegisterAfterHandler(hd ConnAfterCenterHandler) {
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
