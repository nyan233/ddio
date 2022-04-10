package ddio

type eventType int

const (
	LISTENER eventType = 0x01
	CONNECTION eventType = 0x02
)

// MultiEventDispatcher 多路事件派发器
type MultiEventDispatcher struct {
	handlerType eventType
	handler interface{}
	poll *poller
}

func (md *MultiEventDispatcher)RegisterHandler(handler interface{}) bool {
	lhd,ok := handler.(ListenerEventHandler)
	if ok && md.handlerType == LISTENER {
		md.handler = lhd
		return true
	}
	chd,ok := handler.(ConnectionEventHandler)
	if ok && md.handlerType == CONNECTION {
		md.handler = chd
		return true
	}
	return false
}
