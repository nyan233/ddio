package ddio

import (
	"fmt"
	"golang.org/x/sys/unix"
)

// TCPListener Main Reactor
// 负责监听事件
type TCPListener struct {
	event int
}

func NewTCPListener(event int) *TCPListener {
	return &TCPListener{
		event: event,
	}
}

func (T *TCPListener) OnInit(config *NetPollConfig) (*Event, error) {
	event := &Event{}
	fd,err := unix.Socket(unix.AF_INET, unix.SOCK_STREAM,0)
	if err != nil {
		return nil, err
	}
	err = unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_REUSEADDR, 1)
	if err != nil {
		return nil, err
	}
	var addr [4]byte
	copy(addr[:],config.IP.To4())
	err = unix.Bind(fd, &unix.SockaddrInet4{
		Port: config.Port,
		Addr: addr,
	})
	if err != nil {
		return nil, err
	}
	event.sysFd = int32(fd)
	event.event = EventFlags(T.event)
	return event,nil
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

