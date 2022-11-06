//go:build linux || darwin || freebsd

package conn_handler

import (
	"golang.org/x/sys/unix"
	"net"
)

type BeforeConnHandler struct {
}

func (b *BeforeConnHandler) NioRead(fd int, buf []byte) (int, error) {
	return unix.Read(fd, buf)
}

func (b *BeforeConnHandler) NioWrite(fd int, buf []byte) (int, error) {
	return unix.Write(fd, buf)
}

func (b *BeforeConnHandler) Addr(fd int) net.Addr {
	return nil
}

func (b *BeforeConnHandler) Close(fd int) error {
	return unix.Close(fd)
}
