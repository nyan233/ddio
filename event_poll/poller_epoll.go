//go:build linux
package ddio

import (
	"golang.org/x/sys/unix"
	"sync"
	"time"
)

type poller struct {
	*epoll
	onErr func(err error)
	events map[int]Event
	mu *sync.Mutex
}

func NewPoller(onErr func(err error)) *poller {
	ep,err := NewEpoll()
	if err != nil {
		onErr(err)
		return nil
	}
	return &poller{
		ep,
		onErr,
		make(map[int]Event,256),
		&sync.Mutex{},
	}
}

func (p poller) Exec(maxEvent int,timeOut time.Duration) ([]Event,error) {
	events := make([]unix.EpollEvent,maxEvent)
	nEvent, err := p.Wait(events,timeOut)
	if err != nil {
		return nil, err
	}
	events = events[:nEvent]
	stdEvents := make([]Event,nEvent)
	for i := 0; i < nEvent; i++ {
		event := events[i]
		stdEvents[i] = Event{
			sysFd: event.Fd,
			event: func(flags uint32) EventFlags {
				var evFlags EventFlags
				if flags & unix.EPOLLET == unix.EPOLLET {
					evFlags |= 0
				} else if flags & unix.EPOLLIN == unix.EPOLLIN {
					evFlags |= EVENT_READ
				} else if flags & unix.EPOLLOUT == unix.EPOLLOUT {
					evFlags |= EVENT_WRITE
				} else if flags & unix.EPOLLHUP == unix.EPOLLHUP {
					evFlags |= EVENT_CLOSE
				}
				return evFlags
			}(event.Events),
		}
	}
	return stdEvents,nil
}

func (p poller) With(event *Event) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	flags := 0
	if event.event & EVENT_READ == EVENT_READ {
		flags |= unix.EPOLLET | unix.EPOLLIN
	} else if event.event & EVENT_WRITE == EVENT_WRITE {
		flags |= unix.EPOLLET | unix.EPOLLOUT
	} else if event.event & EVENT_CLOSE == EVENT_CLOSE {
		flags |= unix.EPOLLHUP
	}
	vEvent := *event
	event.event = EventFlags(flags)
	err :=p.AddEvent(event)
	if err == nil {
		p.events[int(event.fd())] = vEvent
	}
	return err
}

func (p poller) Modify(event *Event) error {
	return p.ModEvent(event)
}

func (p poller) Cancel(event *Event) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	err := p.DelEvent(event)
	if err != nil {
		delete(p.events, int(event.fd()))
	}
	return err
}

func (p poller) AllEvents() []Event {
	p.mu.Lock()
	defer p.mu.Unlock()
	events := make([]Event,0,len(p.events))
	for _,v := range p.events {
		events = append(events,v)
	}
	return events
}

func (p poller) Exit() error {
	return nil
}