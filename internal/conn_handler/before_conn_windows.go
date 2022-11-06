package conn_handler

import (
	"golang.org/x/sys/windows"
	"net"
)

type BeforeConnHandler struct {
}

func (b *BeforeConnHandler) NioRead(fd int, buf []byte) (int, error) {
	return windows.Read(windows.Handle(fd), buf)
}

func (b *BeforeConnHandler) NioWrite(fd int, buf []byte) (int, error) {
	return windows.Write(windows.Handle(fd), buf)
}

func (b *BeforeConnHandler) Addr(fd int) net.Addr {
	return nil
}

func (b *BeforeConnHandler) Close(fd int) error {
	return windows.Close(windows.Handle(fd))
}
