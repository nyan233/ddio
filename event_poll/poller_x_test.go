package ddio

import (
	"golang.org/x/sys/unix"
	"testing"
	"time"
)

func TestPollerWaitTimeOut(t *testing.T) {
	poller,err := NewPoller()
	if err != nil {
		t.Fatal(err)
	}
	poll := EventLoop(poller)
	//stdInEvent := Event{
	//	sysFd: 4,
	//	event: EVENT_READ,
	//}
	//err = poll.With(stdInEvent)
	//if err != nil {
	//	t.Fatal(err)
	//}
	t1 := time.Now()
	receiver := make([]Event,10)
	_, _ = poll.Exec(receiver, time.Second * 2)
	t2 := time.Now()
	// 测试超时时间的正确性
	if t2.Sub(t1) < time.Second * 1 {
		t.Fatal("poller wait error")
	}
}

func TestAddEvent(t *testing.T) {
	pollerRaw,err := NewPoller()
	poller := EventLoop(pollerRaw)
	if err != nil {
		t.Fatal(err)
	}
	evHandles := 100
	t.Run("AddEvent", func(t *testing.T) {
		for i := 0; i < evHandles; i++ {
			fd, err := unix.Socket(unix.AF_INET, unix.SOCK_STREAM, 0)
			if err != nil {
				t.Fatal(err)
			}
			err = poller.With(Event{
				sysFd: int32(fd),
				event: EVENT_READ,
			})
			if err != nil {
				t.Fatal(err)
			}
		}
		if len(poller.AllEvents()) != evHandles {
			t.Error("add events not equal")
		}
	})
	t.Run("ModifyEvent", func(t *testing.T) {
		allEvents := poller.AllEvents()
		for i := 0; i < evHandles; i++ {
			err := poller.Modify(Event{
				sysFd: allEvents[i].fd(),
				event: EVENT_WRITE,
			})
			if err != nil {
				t.Fatal(err)
			}
		}
		if len(poller.AllEvents()) != evHandles {
			t.Error("modify events not equal")
		}
	})
	t.Run("DeleteEvent", func(t *testing.T) {
		allEvents := poller.AllEvents()
		for i := 0; i < evHandles; i++ {
			err := poller.Cancel(Event{
				sysFd: allEvents[i].fd(),
				event: allEvents[i].Flags(),
			})
			if err != nil {
				t.Fatal(err)
			}
		}
		if len(poller.AllEvents()) != 0 {
			t.Error("add events not equal")
		}
	})
}
