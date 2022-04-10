package ddio

type Event struct {
	sysFd int32
	event EventFlags
}

func (e Event) fd() int32 {
	return e.sysFd
}

func (e Event) Flags() EventFlags {
	return e.event
}

