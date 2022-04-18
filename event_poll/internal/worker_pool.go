package internal

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// WorkerPool 一个简单实现的工作池
type WorkerPool struct {
	// 空闲时的大小
	IdleSize int
	// 超时时间,超过则将工作池的MaxSize - IdleSize
	// 大小的goroutine取消
	Timeout time.Duration
	// 繁忙时的大小
	MaxSize int
	// 处理错误的回调函数
	OnError func(err error)
	// 用于取消所有的goroutine
	// 或者Max - Idle 数量的goroutine
	ctx context.Context
	// 计数器，记录1/10秒钟内添加的任务数量，合适则扩容
	count uint64
	// 接收任务的channel
	task chan func() error
}

func NewWorkerPool(idleSize, maxSize int, timeOut time.Duration, onErr func(err error)) *WorkerPool {
	pool := &WorkerPool{
		IdleSize: idleSize,
		Timeout:  timeOut,
		MaxSize:  maxSize,
		OnError:  onErr,
		ctx:      context.Background(),
		task:     make(chan func() error, maxSize),
	}
	// 等待所有空闲goroutine启动完成
	var waitWg sync.WaitGroup
	waitWg.Add(idleSize)
	// open idle goroutine
	for i := 0; i < idleSize; i++ {
		go func() {
			waitWg.Done()
			for {
				select {
				case fn := <-pool.task:
					err := fn()
					if err != nil {
						pool.OnError(err)
					}
				case <-pool.ctx.Done():
					return
				}
			}
		}()
	}
	waitWg.Wait()
	// open maxSize - idleSize goroutine
	go func() {
		for {
			atomic.StoreUint64(&pool.count, 0)
			time.Sleep(time.Second / 10)
			if !(atomic.LoadUint64(&pool.count) >= 100) {
				continue
			}
			atomic.StoreUint64(&pool.count, 0)
			var wg sync.WaitGroup
			ctx, cancelFn := context.WithTimeout(pool.ctx, pool.Timeout)
			wg.Add(maxSize - idleSize)
			for i := 0; i < maxSize-idleSize; i++ {
				go func() {
					defer wg.Done()
					for {
						select {
						case fn := <-pool.task:
							err := fn()
							if err != nil {
								pool.OnError(err)
							}
						case <-ctx.Done():
							return
						}
					}
				}()
			}
			wg.Wait()
			cancelFn()
		}
	}()
	return pool
}

func (p *WorkerPool) AddTask(fn func() error) {
	p.task <- fn
	atomic.AddUint64(&p.count, 1)
}
