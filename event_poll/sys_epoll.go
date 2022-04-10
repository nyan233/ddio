//go:build linux
package ddio

import (
	"golang.org/x/sys/unix"
	"time"
)

// Epoll Flags
const (
	EPOLLIN = unix.EPOLLIN
	EPOLLET = unix.EPOLLET
	EPOLLONESHOT = unix.EPOLLONESHOT
)


type epoll struct {
	epfd int
}

func NewEpoll() (*epoll,error) {
	fd, err := unix.EpollCreate1(0)
	if err != nil {
		return nil,err
	}
	ep := &epoll{
		epfd:     fd,
	}
	return ep,nil
}

func (e *epoll) AddEvent(ev *Event) error {
	epollEvent := unix.EpollEvent{
		Events: uint32(ev.Flags()),
		Fd:     ev.fd(),
	}
	err := unix.EpollCtl(e.epfd, unix.EPOLL_CTL_ADD, int(epollEvent.Fd), &epollEvent)
	if err != nil {
		return err
	}
	return nil
}

func (e *epoll) Wait(events []unix.EpollEvent,msec time.Duration) (n int,err error) {
	n, err = unix.EpollWait(e.epfd,events, int(msec))
	return
}

func (e *epoll) DelEvent(ev *Event) error {
	epollEvent := unix.EpollEvent{
		Events: uint32(ev.Flags()),
		Fd:     ev.fd(),
	}
	err := unix.EpollCtl(e.epfd, unix.EPOLL_CTL_DEL, int(epollEvent.Fd), &epollEvent)
	if err != nil {
		return err
	}
	return nil
}

func (e *epoll) ModEvent(ev *Event) error {
	epollEvent := unix.EpollEvent{
		Events: uint32(ev.Flags()),
		Fd:     ev.fd(),
	}
	return unix.EpollCtl(e.epfd, unix.EPOLL_CTL_MOD, int(epollEvent.Fd), &epollEvent)
}