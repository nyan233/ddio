package ddio

import (
	"net"
	"time"
)

// ConnectionEventHandler 连接事件处理器
type ConnectionEventHandler interface {
	// OnInit 初始化连接的一些参数
	// 注意: 该方式是在读事件就绪时被触发
	OnInit() ConnConfig

	// OnData 接收到完整数据时触发
	// 接收的数据量可以被设置
	OnData(conn *TCPConn) error

	// OnClose 对端关闭事件触发
	OnClose(ev Event) error

	// OnError 错误事件触发
	OnError(ev Event, err error)
}

// ListenerEventHandler 监听事件处理器
type ListenerEventHandler interface {
	// OnInit 初始化监听者事件处理器时调用的方法
	OnInit(config *NetPollConfig) (*Event, error)
	// OnAccept 有新连接到来时调用的方法
	OnAccept(ev Event) (connFd int, err error)
	// OnClose 客户端退出建立连接阶段调用的方法
	OnClose(ev Event) error
	// OnError 事件循环出错时调用的方法
	OnError(ev Event, err error)
}

// EventLoop 事件循环要实现的接口
type EventLoop interface {
	// Exec 开启事件循环
	// 参数的一些要求，否则可能会出现索引越界等异常
	// Receiver Cap >= maxEvent
	// 各底层Poller最多支持一次性处理 MAX_POLLER_ONCE_EVENTS 个事件
	// nEvent代表发生了多少个事件，如果有错误它为0，同时err != nil
	Exec(receiver []Event, timeOut time.Duration) (nEvent int, err error)

	// Exit 退出事件循环
	Exit() error

	// With 往轮询器中添加事件
	With(event Event) error

	// Modify 修改轮询器中事件的属性
	// 一般用于Epoll OnceShot事件
	Modify(event Event) error

	// Cancel 取消事件的监听
	Cancel(event Event) error

	// AllEvents 获取所有监听的事件
	AllEvents() []Event
}

type Conn interface {
	// TakeReadBytes 拿出所有读取的字节
	TakeReadBytes() []byte
	// WriteBytes 该接口针对小数据量
	WriteBytes(p []byte)
	/*
		RegisterAfterHandler 注册处理后续读数据的处理器。
		注册的after-handler不会影响写缓冲区的数据发送，写缓冲区的数据写入完成
		发生在after-handler被调用之前。如果希望after-handler处理完之后触发写之类
		的操作，可以注册after-result-handler，该处理器在after-handler由于出错或者无错完成的时候被触发
	*/
	// TODO 是否可用于将之后的数据交给ZeroCopy系列函数来处理
	RegisterAfterHandler(ahd AfterHandler, rhd AfterResultHandler)
	// Next 设置了OnDataNBlock时读不到完整的数据时
	// 可以调用该方法接着读取N * Block的内容
	// 调用该方法会使OnData事件重新触发
	Next(nBlock int)
	// Close 关闭连接
	// 非立即关闭，采取延迟关闭的策略
	Close() error
	// Addr 获取兼容net包的Socket Addr
	Addr() net.Addr
	SetDeadLine(deadline time.Duration) error
	SetTimeout(timeout time.Duration) error
}

// Balanced 自定义负载均衡器的接口
type Balanced interface {
	// Name 负载均衡器的名字或者其算法名
	Name() string
	// Target 输入的Seek为ConnectionEventHandler的数量
	// 负载均衡器需要给出一个正确的目标
	// connLen为子Reactor的数量，fd表示新接收连接的文件描述符的值
	Target(connLen, fd int) int
}

// TimerTask 定时器的描述
type TimerTask func(data interface{}, timeOut time.Duration)
