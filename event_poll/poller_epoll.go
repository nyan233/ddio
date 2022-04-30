//go:build linux

package ddio

import (
	"golang.org/x/sys/unix"
	"sync"
	"time"
)

const (
	eP_READ              = unix.EPOLLET | unix.EPOLLIN
	eP_WRITE             = unix.EPOLLET | unix.EPOLLOUT
	eP_LISTENER          = unix.EPOLLIN
	eP_CLOSE             = unix.EPOLLHUP
	eP_ERROR             = unix.EPOLLERR
	ePOLL_ONCE_MAX_EVENT = MAX_POLLER_ONCE_EVENTS
)

type poller struct {
	*epoll
	// Raw Event
	// Fd : Raw Event
	events map[int]EventFlags
	mu     *sync.Mutex
	pool   *sync.Pool
}

func NewPoller() (*poller, error) {
	ep, err := NewEpoll()
	if err != nil {
		return nil, err
	}
	return &poller{
		ep,
		make(map[int]EventFlags, 256),
		&sync.Mutex{},
		&sync.Pool{
			New: func() interface{} {
				return make([]unix.EpollEvent, ePOLL_ONCE_MAX_EVENT)
			},
		},
	}, nil
}

func (p poller) Exec(receiver []Event, timeOut time.Duration) (int, error) {
	events := p.pool.Get().([]unix.EpollEvent)[:cap(receiver)]
	nEvent, err := p.Wait(events, timeOut)
	if err != nil {
		return 0, err
	}
	events = events[:nEvent]
	receiver = receiver[:nEvent]
	p.mu.Lock()
	for i := 0; i < nEvent; i++ {
		event := events[i]
		// 如果发生的是非读写类事件，非读写类事件不能使用ET
		switch {
		case event.Events&eP_ERROR == eP_ERROR:
			break
		case event.Events&eP_CLOSE == eP_CLOSE:
			break
		default:
			// 判断原生事件中有无ET
			// 因为Epoll一次只会触发一个事件
			// 记录的事件可能不只一个，直接转换给上层的感觉就是同时触发了多个事件
			// 主要是为了区分通用的EVENT_LISTENER & EVENT_READ 事件
			if p.events[int(event.Fd)]&unix.EPOLLET == unix.EPOLLET {
				event.Events |= EPOLLET
			}
		}
		receiver[i] = Event{
			sysFd: event.Fd,
			event: epollToEvent(int(event.Events)),
		}
	}
	p.mu.Unlock()
	p.pool.Put(events)
	return nEvent, nil
}

func (p poller) With(event Event) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	flags := eventToEpoll(event.event)
	event.event = EventFlags(flags)
	err := p.AddEvent(&event)
	if err == nil {
		p.events[int(event.fd())] = event.Flags()
	}
	return err
}

func (p poller) Modify(event Event) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	flags := eventToEpoll(event.Flags())
	event.event = EventFlags(flags)
	err := p.ModEvent(&event)
	if err == nil {
		p.events[int(event.fd())] = event.Flags()
	}
	return err
}

func (p poller) Cancel(event Event) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	err := p.DelEvent(&event)
	if err != nil {
		delete(p.events, int(event.fd()))
	}
	return err
}

func (p poller) AllEvents() []Event {
	p.mu.Lock()
	defer p.mu.Unlock()
	events := make([]Event, 0, len(p.events))
	for k, v := range p.events {
		events = append(events, Event{
			sysFd: int32(k),
			event: epollToEvent(int(v)),
		})
	}
	return events
}

func (p poller) Exit() error {
	return nil
}

// Utils
func eventToEpoll(flags EventFlags) int {
	var epFlags EventFlags
	if flags&EVENT_READ == EVENT_READ {
		epFlags |= eP_READ
	} else if flags&EVENT_WRITE == EVENT_WRITE {
		epFlags |= eP_WRITE
	} else if flags&EVENT_CLOSE == EVENT_CLOSE {
		epFlags |= eP_CLOSE
	} else if flags&EVENT_LISTENER == EVENT_LISTENER {
		epFlags |= eP_LISTENER
	} else if flags&EVENT_ERROR == EVENT_ERROR {
		epFlags |= eP_ERROR
	}
	return int(epFlags)
}

func epollToEvent(epollEvent int) EventFlags {
	var flags EventFlags
	if epollEvent&eP_READ == eP_READ {
		flags |= EVENT_READ
	} else if epollEvent&eP_WRITE == eP_WRITE {
		flags |= EVENT_WRITE
	} else if epollEvent&eP_LISTENER == eP_LISTENER {
		flags |= EVENT_LISTENER
	} else if epollEvent&eP_CLOSE == eP_CLOSE {
		flags |= EVENT_CLOSE
	} else if epollEvent&eP_ERROR == eP_ERROR {
		flags |= EVENT_ERROR
	}
	return flags
}
