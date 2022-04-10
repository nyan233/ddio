package ddio

import "time"

// ConnectionEventHandler 连接事件处理器
type ConnectionEventHandler interface {
	// OnRead 读就绪事件触发
	OnRead(readBuffer, writeBuffer []byte, flags EventFlags) error

	// OnWrite 写就绪事件触发
	OnWrite(writeBuffer []byte, flags EventFlags) error

	// OnClose 对端关闭事件触发
	OnClose(ev Event) error

	// OnError 错误事件触发
	OnError(ev Event, err error)
}

// ListenerEventHandler 监听事件处理器
type ListenerEventHandler interface {
	OnAccept(ev Event) (connFd int, err error)
	OnClose(ev Event) error
	OnError(ev Event,err error)
}

// EventLoop 事件循环要实现的接口
type EventLoop interface {
	// Exec 开启事件循环
	Exec(maxEvent int,timeOut time.Duration) ([]Event,error)

	// With 往轮询器中添加事件
	With(event *Event) error

	// Modify 修改轮询器中事件的属性
	// 一般用于Epoll OnceShot事件
	Modify(event *Event) error

	// Cancel 取消事件的监听
	Cancel(event *Event) error

	// AllEvents 获取所有监听的事件
	AllEvents() []Event
}
