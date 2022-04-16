//go:build linux

package ddio

import (
	"golang.org/x/sys/unix"
	"sync"
	"time"
)

const (
	eV_READ     = unix.EPOLLET | unix.EPOLLIN
	eV_WRITE    = unix.EPOLLET | unix.EPOLLOUT
	eV_LISTENER = unix.EPOLLIN
	eV_CLOSE    = unix.EPOLLHUP
)

type poller struct {
	*epoll
	// Raw Event
	// Fd : Raw Event
	events map[int]EventFlags
	mu     *sync.Mutex
}

func NewPoller() (*poller,error) {
	ep, err := NewEpoll()
	if err != nil {
		return nil,err
	}
	return &poller{
		ep,
		make(map[int]EventFlags, 256),
		&sync.Mutex{},
	},nil
}

func (p poller) Exec(maxEvent int, timeOut time.Duration) ([]Event, error) {
	events := make([]unix.EpollEvent, maxEvent)
	nEvent, err := p.Wait(events, timeOut)
	if err != nil {
		return nil, err
	}
	events = events[:nEvent]
	stdEvents := make([]Event, nEvent)
	for i := 0; i < nEvent; i++ {
		event := events[i]
		p.mu.Lock()
		stdEvents[i] = Event{
			sysFd: event.Fd,
			event: epollToEvent(int(p.events[int(event.Fd)])),
		}
		p.mu.Unlock()
	}
	return stdEvents, nil
}

func (p poller) With(event *Event) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	flags := eventToEpoll(event.event)
	event.event = EventFlags(flags)
	err := p.AddEvent(event)
	if err == nil {
		p.events[int(event.fd())] = event.Flags()
	}
	return err
}

func (p poller) Modify(event *Event) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	flags := eventToEpoll(event.Flags())
	event.event = EventFlags(flags)
	err := p.ModEvent(event)
	if err == nil {
		p.events[int(event.fd())] = event.Flags()
	}
	return err
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
		epFlags |= eV_READ
	} else if flags&EVENT_WRITE == EVENT_WRITE {
		epFlags |= eV_WRITE
	} else if flags&EVENT_CLOSE == EVENT_CLOSE {
		epFlags |= eV_CLOSE
	} else if flags&EVENT_LISTENER == EVENT_LISTENER {
		epFlags |= eV_LISTENER
	}
	return int(epFlags)
}

func epollToEvent(epollEvent int) EventFlags {
	var flags EventFlags
	if epollEvent&eV_READ == eV_READ {
		flags |= EVENT_READ
	} else if epollEvent&eV_WRITE == eV_WRITE {
		flags |= EVENT_WRITE
	} else if epollEvent&eV_LISTENER == eV_LISTENER {
		flags |= EVENT_LISTENER
	} else if epollEvent&eV_CLOSE == eV_CLOSE {
		flags |= EVENT_CLOSE
	}
	return flags
}
