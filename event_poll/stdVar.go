package ddio

import "errors"

// 这里描述了一些通用的提示值

// EventFlags 通用的事件掩码
type EventFlags int

const (
	EVENT_READ  EventFlags = 0x01 // 监听可读事件
	EVENT_WRITE EventFlags = 0x02 // 监听可写事件
	EVENT_CLOSE EventFlags = 0x10 // 监听连接关闭事件
)

// 处理错误的标准函数
type ErrorHandler func(err error)

// 一些错误
var (
	ErrorEpollClosed = errors.New("epoll is closed")
)