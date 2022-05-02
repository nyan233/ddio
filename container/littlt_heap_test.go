package container

import (
	"errors"
	"testing"
	"time"
)

func TestLittleHeap(t *testing.T) {
	n := 1 << 5
	lHeap := NewLittleHeap(n)
	for i := 1; i <= n; i++ {
		lHeap.Insert(TimeoutElem{
			TimeOut: time.Duration(i),
			Data:    nil,
		})
	}
	if lHeap.Size() != n {
		t.Fatal(errors.New("little heap size is not equal"))
	}
	// insert
	lHeap.Insert(TimeoutElem{
		TimeOut: time.Duration(n / 2),
		Data:    nil,
	})
	t.Log(lHeap)
	for i := 0; i < n / 2; i++ {
		lHeap.DelTop()
	}
	t.Log(lHeap)
}
