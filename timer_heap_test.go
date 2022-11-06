package ddio

import (
	"math/rand"
	"sync"
	"testing"
	"time"
)

func TestTimer(t *testing.T) {
	now := time.Duration(time.Now().UnixNano())
	// 添加100万个超时事件
	rand.Seed(time.Now().UnixNano())
	nEvent := 1000000
	timer := newDDTimer(now, time.Millisecond, 64, 1024*1024, nEvent)
	var wg sync.WaitGroup
	mu := sync.Mutex{}
	wg.Add(nEvent)
	// 统计生成的最大超时时间
	var maxTimeOut time.Duration
	var fn TimerTask = func(data interface{}, timeOut time.Duration) {
		mu.Lock()
		if timeOut > maxTimeOut {
			maxTimeOut = timeOut
		}
		wg.Done()
		mu.Unlock()
	}
	for i := 0; i < nEvent; i++ {
		timer.AddTimer(false, time.Millisecond*time.Duration(rand.Intn(1000)+1), 10, fn)
	}
	// 触发这些超时事件
	wg.Wait()
	t.Log(maxTimeOut - now)
}

func TestStdTimer(t *testing.T) {
	timer := time.NewTimer(time.Second * 1)
	<-timer.C
}
