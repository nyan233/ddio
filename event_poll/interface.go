package ddio

import (
	"net"
	"time"
)

// ConnectionEventHandler 连接事件处理器
type ConnectionEventHandler interface {
	// OnData 接收到数据时触发
	OnData(conn Conn) error

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

type Conn interface {
	// TakeReadBytes 拿出所有读取的字节
	TakeReadBytes() []byte
	// TakeWriteBuffer 拿出Conn绑定的写入缓冲
	// 该接口针对较大的数据使用
	TakeWriteBuffer() *[]byte
	// GrowWriteBuffer 扩容缓冲区
	GrowWriteBuffer(buf *[]byte, nCap int) bool
	// WriteBytes 该接口针对小数据量
	WriteBytes(p []byte)
	// Close 关闭连接
	// 非立即关闭，采取延迟关闭的策略
	Close() error
	// Addr 获取兼容net包的Socket Addr
	Addr() net.Addr
	SetDeadLine(deadline time.Time) error
	SetTimeout(timeout time.Duration) error
}


// Balanced 自定义负载均衡器的接口
type Balanced interface {
	// Name 负载均衡器的名字或者其算法名
	Name() string
	// Target 输入的Seek为ConnectionEventHandler的数量
	// 负载均衡器需要给出一个正确的目标
	Target(seek int) int
}