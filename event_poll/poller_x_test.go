package ddio

import (
	"testing"
	"time"
)

func TestPollerWaitTimeOut(t *testing.T) {
	poller,err := NewPoller()
	if err != nil {
		t.Fatal(err)
	}
	poll := EventLoop(poller)
	stdInEvent := Event{
		sysFd: 4,
		event: EVENT_READ,
	}
	err = poll.With(stdInEvent)
	if err != nil {
		t.Fatal(err)
	}
	t1 := time.Now()
	receiver := make([]Event,10)
	_, _ = poll.Exec(receiver, time.Second * 2)
	t2 := time.Now()
	// 测试超时时间的正确性
	if t2.Sub(t1) < time.Second * 1 {
		t.Fatal("poller wait error")
	}
}
