package ddio

// 这里描述了一些通用的提示值

// EventFlags 通用的事件掩码
type EventFlags int

const (
	EVENT_READ  int = 0x01 // 监听可读事件
	EVENT_WRITE int = 0x02 // 监听可写事件
	EVENT_CLOSE int = 0x10 // 监听连接关闭事件
)
