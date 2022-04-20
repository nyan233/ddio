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
		pool:   &sync.Pool{
			New: func() interface{} {
				return make([]unix.Kevent_t,MAX_POLLER_ONCE_EVENTS)
			},
		},
	}, nil
}

// Exec Kqueue 没有像Epoll那样可以监听Close事件，也没有ET/LT
// 之分，所以Close事件不会得到响应。
func (p poller) Exec(receiver []Event, timeOut time.Duration) (nEvent int, err error) {
	kEvents := p.pool.Get().([]unix.Kevent_t)
	readyN, err := p.Wait(kEvents, timeOut)
	if err != nil {
		return 0, err
	}
	kEvents = kEvents[:readyN]
	receiver = receiver[:readyN]
	p.mu.Lock()
	for k,v := range kEvents {
		flags := kqueueToEvent(int(v.Filter))
		if flags == kQ_READ && p.events[int(v.Ident)] == EVENT_LISTENER {
			flags = EVENT_LISTENER
		}
		receiver[k] = Event{
			sysFd: int32(v.Ident),
			event: flags,
		}
	}
	p.mu.Unlock()
	return readyN,err
}

func (p poller) Exit() error {
	return unix.Close(p.kqfd)
}

func (p poller) With(event Event) error {
	oldFlags := event.Flags()
	event.event = EventFlags(eventToKqueue(event.Flags()))
	err := p.AddEvent(&event)
	p.mu.Lock()
	if err != nil {
		delete(p.events, int(event.fd()))
		p.mu.Unlock()
		return err
	}
	p.events[int(event.fd())] = oldFlags
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
	events := make([]Event,0,len(p.events))
	for k, v := range p.events {
		events = append(events,Event{
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
	case kQ_WRITE:
		flags = EVENT_WRITE
	case kQ_ERROR:
		flags = EVENT_ERROR
	}
	return flags
}

func eventToKqueue(event EventFlags) int {
	var flags int
	switch  {
	case event & EVENT_READ == EVENT_READ:
		flags = kQ_READ
		break
	case event & EVENT_WRITE == EVENT_WRITE:
		flags = kQ_WRITE
		break
	case event & EVENT_LISTENER == EVENT_LISTENER:
		flags = kQ_READ
		break
	}
	return flags
}
