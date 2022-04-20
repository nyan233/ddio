//go:build darwin  || freebsd

package ddio

import (
	"golang.org/x/sys/unix"
	"time"
)

type kqueue struct {
	kqfd int
}

func NewKqueue() (*kqueue, error) {
	kq := &kqueue{}
	kqfd,err := unix.Kqueue()
	if err != nil {
		return nil, err
	}
	kq.kqfd = kqfd
	return kq,nil
}

func (k *kqueue) AddEvent(ev *Event) error {
	var events [1]unix.Kevent_t
	events[0] = unix.Kevent_t{
		Ident:  uint64(ev.fd()),
		Filter: int16(ev.Flags()),
		Flags:  unix.EV_ADD | unix.EV_ENABLE,
		Fflags: 0,
		Data:   0,
		Udata: nil,
	}
	_,err := unix.Kevent(k.kqfd,events[:],nil,nil)
	return err
}



func (k *kqueue) ModifyEvent(ev *Event) error {
	return k.AddEvent(ev)
}

func (k *kqueue) DelEvent(ev *Event) error {
	var events [1]unix.Kevent_t
	events[0] = unix.Kevent_t{
		Ident:  uint64(ev.fd()),
		Filter: int16(ev.Flags()),
		Flags:  unix.EV_DELETE | unix.EV_ONESHOT,
		Fflags: 0,
		Data:   0,
		Udata: nil,
	}
	_,err := unix.Kevent(k.kqfd,events[:],nil,nil)
	return err
}

func (k *kqueue) Wait(events []unix.Kevent_t,timeOut time.Duration) (n int,err error) {
	timeSpec := &unix.Timespec{
		Sec:  timeOut.Milliseconds() / 1000,
		Nsec: 0,
	}
	return unix.Kevent(k.kqfd,nil,events,timeSpec)
}