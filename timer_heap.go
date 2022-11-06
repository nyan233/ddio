package ddio

import (
	"errors"
	"github.com/zbh255/nyan/container"
	"github.com/zbh255/nyan/event_poll/internal"
	"sync"
	"sync/atomic"
	"time"
)

var (
	// ErrTimerClosed 时间堆已经被关闭
	ErrTimerClosed = errors.New("timer is closed")
	// ErrTimerFull 时间堆的定时任务数量已经到达设定的最大值
	ErrTimerFull = errors.New("time is full")
)

/*
基于小顶堆的定时器的实现
该实现是线程安全的
TODO 该定时器的实现在百万/千万级别的任务处理时延迟明显，考虑用工作池来优化
*/
type ddTimer struct {
	mu sync.Mutex
	// 底层的小顶堆
	lHeap *container.LittleHeap
	// 触发的阈值
	click time.Duration
	// std Ticker
	ticker *time.Ticker
	// 关闭标志
	closed int64
	// 活跃定时任务的最大数量
	maxSize int
	// 绑定的工作池
	wp *internal.WorkerPool
}

type timerData [2]interface{}

func newDDTimer(initTime time.Duration, click time.Duration, wpSize, wpBufSize, maxSize int) *ddTimer {
	ticker := time.NewTicker(click)
	wp := internal.NewWorkerPool(wpSize, wpBufSize, timerHandle, func(_ error) {
		return
	})
	ddt := &ddTimer{
		lHeap:   container.NewLittleHeap(1 << 8),
		click:   initTime,
		ticker:  ticker,
		wp:      wp,
		maxSize: maxSize,
	}
	go ddt.OpenTimerLoop()
	return ddt
}

func timerHandle(data interface{}) error {
	elem := data.(container.TimeoutElem)
	td := elem.Data.(timerData)
	uData := td[0]
	uTimer := td[1].(TimerTask)
	uTimer(uData, elem.TimeOut)
	return nil
}

// AddTimer isAbsTimeOut == true则意味着这个超时值是绝对时间
func (t *ddTimer) AddTimer(isAbsTimeOut bool, timeOut time.Duration, data interface{}, timer TimerTask) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if atomic.LoadInt64(&t.closed) == 1 {
		return ErrTimerClosed
	}
	if t.maxSize <= t.lHeap.Size() {
		return ErrTimerFull
	}
	t.lHeap.Insert(container.TimeoutElem{
		TimeOut: func() time.Duration {
			if isAbsTimeOut {
				return timeOut
			}
			return t.click + timeOut
		}(),
		Data: timerData{
			data,
			timer,
		},
	})
	return nil
}

// Click 如果要检查多个过期Timer，调用者需要将timeOut设置为0重复检查
// 这是从内存复用的角度考虑
func (t *ddTimer) Click(timeOut time.Duration) (elem container.TimeoutElem) {
	if t.lHeap.IsEmpty() {
		return container.TimeoutElem{}
	}
	t.click += timeOut
	topTimeOut := t.lHeap.Peek().TimeOut
	// 到点则触发到期
	if t.click >= topTimeOut {
		elem = t.lHeap.DelTop()
	}
	return
}

func (t *ddTimer) ResetClick() {
	t.mu.Lock()
	t.click = 0
	t.mu.Unlock()
}

func (t *ddTimer) Close() error {
	if !atomic.CompareAndSwapInt64(&t.closed, 0, 1) {
		return ErrTimerClosed
	}
	t.ticker.Stop()
	t.wp.Stop()
	return nil
}

// OpenTimerLoop 循环处理到期事件
func (t *ddTimer) OpenTimerLoop() {
	for {
		select {
		case <-t.ticker.C:
			t.mu.Lock()
			for elem := t.Click(time.Millisecond); elem.Data != nil; {
				t.wp.PushTask(elem)
				elem = t.Click(0)
			}
			t.mu.Unlock()
		default:
			if atomic.LoadInt64(&t.closed) == 1 {
				return
			}
		}
	}
}
