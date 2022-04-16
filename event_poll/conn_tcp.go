package ddio

import (
	"net"
	"time"
)

type TCPConn struct {
	rawFd int
	rBytes []byte
	wBytes []byte
	memPool *BufferPool
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

func (T *TCPConn) GrowWriteBuffer(buf *[]byte, nCap int) bool {
	return T.memPool.Grow(buf,nCap)
}

func (T *TCPConn) WriteBytes(p []byte) {
	for len(p) + len(T.wBytes) > cap(T.wBytes) {
		if !T.memPool.Grow(&T.wBytes,(cap(T.wBytes)/int(T.memPool.block)) * 2) {
			tmp := make([]byte,0,cap(T.wBytes) * 2)
			tmp = append(tmp,T.wBytes...)
			T.memPool.FreeBuffer(&T.wBytes)
			T.wBytes = tmp
		}
	}
	T.wBytes = append(T.wBytes,p...)
}

func (T *TCPConn) Close() error {
	panic("implement me")
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
