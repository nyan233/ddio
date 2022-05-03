package ddio

import (
	"math/rand"
	"testing"
	"time"
)

func TestTimer(t *testing.T) {
	now := time.Duration(time.Now().UnixNano())
	timer := newDDTimer(now,time.Millisecond)
	// 添加10万个超时事件
	rand.Seed(time.Now().UnixNano())
	nEvent := 100000
	nTimeOut := 0
	// 统计生成的最大超时时间
	var maxTimeOut time.Duration
	var fn TimerTask = func(data interface{}, timeOut time.Duration) {
		nTimeOut++
		if timeOut > maxTimeOut {
			maxTimeOut = timeOut
		}
	}
	for i := 0; i < nEvent; i++ {
		timer.AddTimer(false,time.Millisecond * time.Duration(rand.Intn(1000) + 1),10,fn)
	}
	// 触发这些超时事件
	for !(nTimeOut == nEvent) {}
	t.Log(now)
	t.Log(maxTimeOut)
}

func TestStdTimer(t *testing.T) {
	timer := time.NewTimer(time.Second * 1)
	<-timer.C
}
