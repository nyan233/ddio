package ddio

import (
	"errors"
	"time"
)

// 这里描述了一些通用的提示值

// EventFlags 通用的事件掩码
type EventFlags int

const (
	EVENT_READ     EventFlags = 0x01   // 监听可读事件
	EVENT_WRITE    EventFlags = 0x10   // 监听可写事件
	EVENT_CLOSE    EventFlags = 0x100  // 监听连接关闭事件
	EVENT_LISTENER EventFlags = 0x04   // 监听连接建立事件
	EVENT_ERROR    EventFlags = 0x1000 // 监听错误事件
)

const (
	// MAX_MASTER_LOOP_SIZE 负责监听接收新连接的主Reactor的goroutine最大数量
	MAX_MASTER_LOOP_SIZE = 32
	// MAX_SLAVE_LOOP_SIZE 主Reactor绑定的负责处理连接事件的从Reactor的goroutine最大数量
	MAX_SLAVE_LOOP_SIZE = 64
	// MAX_POLLER_ONCE_EVENTS 各底层Poller一次最多响应的就绪事件
	MAX_POLLER_ONCE_EVENTS = 1024
	// EVENT_LOOP_SLEEP 事件循环阻塞等待就绪事件的默认超时时间
	EVENT_LOOP_SLEEP time.Duration = time.Millisecond
)

const (
	TCP_V4 = 0x80
	TCP_V6 = 0x90
)

// 一些错误
var (
	ErrorEpollClosed = errors.New("epoll is closed")
	ErrRead          = errors.New("read error: ")
	ErrWrite         = errors.New("write error: ")
)

var (
	DefaultConfig = ConnConfig{
		OnDataNBlock:                     1,
		MaxReadSysCallNumberOnEventLoop:  1024,
		MaxWriteSysCallNumberOnEventLoop: 1024,
	}
)

// NewBalance 派生负载均衡器的工厂方法
type NewBalance func() Balanced

// AfterHandler 是一对回调函数,bool用于指示在后续的声明周期中是否还需要调用
// 否则立即调用result-handler，用于Non-Block IO
type AfterHandler func(fd int) (error, bool)
type AfterResultHandler func(fd int, err error) error
