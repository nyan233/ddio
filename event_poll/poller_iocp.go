//go:build windows
package ddio

type poller struct {
	*iocp
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

