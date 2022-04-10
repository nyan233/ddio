package ddio

import (
	"fmt"
	"golang.org/x/sys/unix"
)

// TCPListener Main Reactor
// 负责监听事件
type TCPListener struct {

}

func (T *TCPListener) OnAccept(ev Event) (connFd int, err error) {
	connFd, _, err = unix.Accept(int(ev.fd()))
	if err != nil {
		return 0, err
	}
	err = unix.SetNonblock(connFd,true)
	return connFd,err
}

func (T *TCPListener) OnClose(ev Event) error {
	return unix.Close(int(ev.fd()))
}

func (T *TCPListener) OnError(ev Event,err error) {
	logger.ErrorFromString(fmt.Sprintf("TcpListener On Error: %s",err))
}

