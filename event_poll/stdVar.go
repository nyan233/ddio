package ddio

import "errors"

// 这里描述了一些通用的提示值

// EventFlags 通用的事件掩码
type EventFlags int

const (
	EVENT_READ     EventFlags = 0x01                     // 监听可读事件
	EVENT_WRITE    EventFlags = 0x10                     // 监听可写事件
	EVENT_CLOSE    EventFlags = 0x100                    // 监听连接关闭事件
	EVENT_LISTENER EventFlags = 0x04                     // 监听连接建立事件
)


// 一些错误
var (
	ErrorEpollClosed = errors.New("epoll is closed")
	ErrRead = errors.New("read error: ")
	ErrWrite = errors.New("write error: ")
)

// NewBalance 派生负载均衡器的工厂方法
type NewBalance func() Balanced