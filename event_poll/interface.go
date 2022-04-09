package ddio

// EventHandler 事件处理器
type EventHandler interface {
	// OnRead 读就绪事件触发
	OnRead(readBuffer,writeBuffer []byte, flags EventFlags) error

	// OnWrite 写就绪事件触发
	OnWrite(writeBuffer []byte, flags EventFlags) error

	// OnClose 对端关闭事件触发
	OnClose(flags EventFlags) error

	// OnError 错误事件触发
	OnError(flags EventFlags,err error)
}

// EventLoop 事件循环要实现的接口
type EventLoop interface {
	// With 往轮询器中添加事件
	With(event *Event) error

	// Modify 修改轮询器中事件的属性
	// 一般用于Epoll OnceShot事件
	Modify(event *Event) error

	// Cancel 取消事件的监听
	Cancel(event *Event) error
}