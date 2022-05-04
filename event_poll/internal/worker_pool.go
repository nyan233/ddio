package internal

import (
	"context"
	"runtime"
)

// WorkerPool TODO 重构工作池的代码
// WorkerPool 一个简单实现的工作池
type WorkerPool struct {
	// 繁忙时的大小
	MaxSize int
	// 处理错误的回调函数
	OnError func(err error)
	// 用于取消所有的goroutine
	cancel context.CancelFunc
	// 接收任务的channel
	task chan interface{}
	// 注册的处理函数
	handleFn func(data interface{}) error
}

func NewWorkerPool(size,bufSize int, handleFn func(_ interface{}) error,onErr func(_ error)) *WorkerPool {
	ctx,cancel := context.WithCancel(context.Background())
	wp := &WorkerPool{
		MaxSize:  size,
		OnError:  onErr,
		cancel:   cancel,
		task:     make(chan interface{},bufSize),
		handleFn: handleFn,
	}
	wp.openGs(ctx)
	return wp
}

func (p *WorkerPool) openGs(ctx context.Context) {
	for i := 0; i < p.MaxSize; i++ {
		go func() {
			for {
				select {
				case data := <- p.task:
					err := p.handleFn(data)
					if err != nil {
						p.OnError(err)
					}
				case <-ctx.Done():
					return
				default:
					runtime.Gosched()
				}
			}
		}()
	}
}

func (p *WorkerPool) Stop() {
	p.cancel()
}

func (p *WorkerPool) PushTask(data interface{}) {
	p.task <- data
}



