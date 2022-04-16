package conn_handler

type BeforeConnHandler struct {

}

func (b *BeforeConnHandler) NioRead(fd int,buf []byte) (int, error) {
	return unix.Read(fd,buf)
}

func (b *BeforeConnHandler) NioWrite(fd int,buf []byte) (int, error) {
	return unix.Write(fd,buf)
}

func (b *BeforeConnHandler) Addr(fd int) net.Addr {
	unix.Getsockname()
}