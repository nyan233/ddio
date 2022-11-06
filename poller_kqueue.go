//go:build darwin || freebsd

package ddio

import (
	"golang.org/x/sys/unix"
	"sync"
	"time"
)

const (
	kQ_READ  = unix.EVFILT_READ
	kQ_WRITE = unix.EVFILT_WRITE
	kQ_ERROR = unix.EV_ERROR
)

type poller struct {
	*kqueue
	// User Event
	// Fd : User Event
	// Poller-Kqueue中这个字段的含义跟Poller-Epoll中不同
	// 该字段存储的是用户定义的Event-Flags
	events map[int]EventFlags
	mu     *sync.Mutex
	pool   *sync.Pool
}

func NewPoller() (*poller, error) {
	kqPoller, err := NewKqueue()
	if err != nil {
		return nil, err
	}
	return &poller{
		kqueue: kqPoller,
		events: make(map[int]EventFlags, 256),
		mu:     &sync.Mutex{},
		pool: &sync.Pool{
			New: func() interface{} {
				return make([]unix.Kevent_t, MAX_POLLER_ONCE_EVENTS)
			},
		},
	}, nil
}

// Exec Kqueue 没有像Epoll那样可以监听Close事件，也没有ET/LT
// 之分，所以Close事件不会得到响应。
func (p poller) Exec(receiver []Event, timeOut time.Duration) (nEvent int, err error) {
	kEvents := p.pool.Get().([]unix.Kevent_t)[:cap(receiver)]
	readyN, err := p.Wait(kEvents, timeOut)
	if err != nil {
		return 0, err
	}
	kEvents = kEvents[:readyN]
	receiver = receiver[:readyN]
	p.mu.Lock()
	for k, v := range kEvents {
		flags := kqueueToEvent(int(v.Filter))
		if flags == kQ_READ && p.events[int(v.Ident)] == EVENT_LISTENER {
			flags = EVENT_LISTENER
		}
		// Error Or Eof
		if v.Flags&kQ_ERROR != 0 || v.Flags&unix.EV_EOF != 0 {
			flags |= EVENT_ERROR
		}
		receiver[k] = Event{
			sysFd: int32(v.Ident),
			event: flags,
		}
	}
	p.mu.Unlock()
	p.pool.Put(kEvents)
	return readyN, err
}

func (p poller) Exit() error {
	return unix.Close(p.kqfd)
}

func (p poller) With(event Event) error {
	p.mu.Lock()
	// 思路取自: github.com/Allenxuxu/gev
	// Link : https://github.com/Allenxuxu/gev/blob/7ac1dc183d41d1503378a0b9edc2cdc180be9487/poller/kqueue.go#L126
	oldEv, ok := p.events[int(event.fd())]
	var kEvents []unix.Kevent_t
	if ok {
		kEvents = append(kEvents, unix.Kevent_t{
			Ident:  uint64(event.fd()),
			Filter: int16(eventToKqueue(oldEv)),
			Flags:  unix.EV_DELETE | unix.EV_ONESHOT,
			Fflags: 0,
			Data:   0,
			Udata:  nil,
		})
	}
	kEvents = append(kEvents, unix.Kevent_t{
		Ident:  uint64(event.fd()),
		Filter: int16(eventToKqueue(event.Flags())),
		Flags:  unix.EV_ADD | unix.EV_ENABLE,
		Fflags: 0,
		Data:   0,
		Udata:  nil,
	})
	_, err := unix.Kevent(p.kqfd, kEvents, nil, nil)
	if err != nil {
		delete(p.events, int(event.fd()))
		p.mu.Unlock()
		return err
	}
	p.events[int(event.fd())] = event.Flags()
	p.mu.Unlock()
	return nil
}

func (p poller) Modify(event Event) error {
	return p.With(event)
}

func (p poller) Cancel(event Event) error {
	p.mu.Lock()
	delete(p.events, int(event.fd()))
	p.mu.Unlock()
	event.event = EventFlags(eventToKqueue(event.Flags()))
	return p.DelEvent(&event)
}

func (p poller) AllEvents() []Event {
	p.mu.Lock()
	events := make([]Event, 0, len(p.events))
	for k, v := range p.events {
		events = append(events, Event{
			sysFd: int32(k),
			event: v,
		})
	}
	p.mu.Unlock()
	return events
}

func kqueueToEvent(event int) EventFlags {
	var flags EventFlags
	switch event {
	case kQ_READ:
		flags = EVENT_READ
		break
	case kQ_WRITE:
		flags = EVENT_WRITE
		break
	}
	return flags
}

func eventToKqueue(event EventFlags) int {
	var flags int
	switch {
	case event&EVENT_READ == EVENT_READ:
		flags = kQ_READ
		break
	case event&EVENT_WRITE == EVENT_WRITE:
		flags = kQ_WRITE
		break
	case event&EVENT_LISTENER == EVENT_LISTENER:
		flags = kQ_READ
		break
	}
	return flags
}
