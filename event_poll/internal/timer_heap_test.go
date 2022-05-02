package internal

import (
	"github.com/zbh255/nyan/container"
	"math/rand"
	"testing"
	"time"
)

func TestTimer(t *testing.T) {
	now := time.Duration(time.Now().UnixNano())
	timer := NewDDTimer(now)
	// 添加1万个超时事件
	rand.Seed(time.Now().UnixNano())
	nEvent := 10000
	for i := 0; i < nEvent; i++ {
		timer.AddTimer(container.TimeoutElem{
			TimeOut: now + time.Millisecond * time.Duration(rand.Intn(1000) + 1),
			Data:    i+1,
		})
	}
	t.Log(timer.lHeap)
	// 触发这些超时事件
	nTimeOut := 0
	for {
		tick := time.Millisecond * 2
		time.Sleep(time.Millisecond * 2)
		for timer.Click(tick).Data != nil {
			nTimeOut++
			tick = 0
		}
		if nTimeOut == nEvent {
			break
		}
	}
}

func TestStdTimer(t *testing.T) {
	timer := time.NewTimer(time.Second * 3)
	<-timer.C
}
