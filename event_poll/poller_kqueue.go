//go:build darwin || openbsd || freebsd
package ddio

type poller struct {
	*kqueue
}

func (p poller) With(event *Event) error {
	panic("implement me")
}

func (p poller) Modify(event *Event) error {
	panic("implement me")
}

func (p poller) Cancel(event *Event) error {
	panic("implement me")
}
