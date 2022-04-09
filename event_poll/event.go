package ddio

// Event 用于描述Event数据结构
type Event struct {
	sysFd int32
	event EventFlags
	handler EventHandler
}

func (e Event) fd() int32 {
	return e.sysFd
}

func (e Event) Flags() EventFlags {
	return e.event
}

func (e Event) Handler() EventHandler {
	return e.handler
}

func NewEvent(sysFd int32, event EventFlags, handler EventHandler) *Event {
	return &Event{
		sysFd:   sysFd,
		event:   event,
		handler: handler,
	}
}