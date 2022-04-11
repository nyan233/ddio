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
	// OnInit 初始化监听者事件处理器时调用的方法
	OnInit(config *NetPollConfig) (*Event,error)
	// OnAccept 有新连接到来时调用的方法
	OnAccept(ev Event) (connFd int, err error)
	// OnClose 客户端退出建立连接阶段调用的方法
	OnClose(ev Event) error
	// OnError 事件循环出错时调用的方法
	OnError(ev Event,err error)
}

// EventLoop 事件循环要实现的接口
type EventLoop interface {
	// Exec 开启事件循环
	Exec(maxEvent int,timeOut time.Duration) ([]Event,error)

	// Exit 退出事件循环
	Exit() error

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
