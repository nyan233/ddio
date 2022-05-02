package internal

import (
	"github.com/zbh255/nyan/container"
	"time"
)

/*
	基于小顶堆的定时器的实现
*/
type ddTimer struct {
	// 底层的小顶堆
	lHeap *container.LittleHeap
	// 触发的阈值
	click time.Duration
}

func NewDDTimer(initClick time.Duration) *ddTimer {
	return &ddTimer{
		lHeap: container.NewLittleHeap(1 << 8),
		click: initClick,
	}
}

func (t *ddTimer) AddTimer(timer container.TimeoutElem) {
	t.lHeap.Insert(timer)
}

// Click 如果要检查多个过期Timer，调用者需要将timeOut设置为0重复检查
// 这是从内存复用的角度考虑
func (t *ddTimer) Click(timeOut time.Duration) (elem container.TimeoutElem) {
	t.click += timeOut
	topTimeOut := t.lHeap.Peek().TimeOut
	// 到点则触发到期
	if t.click >= topTimeOut {
		elem = t.lHeap.DelTop()
	}
	return
}

func (t *ddTimer) ResetClick() {
	t.click = 0
}