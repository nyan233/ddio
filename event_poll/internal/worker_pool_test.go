package internal

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestWorkerPool(t *testing.T) {
	onErr := func(err error) {
		t.Error(err)
	}
	pool := NewWorkerPool(1200, 1500, time.Minute, onErr)
	var wg sync.WaitGroup
	wg.Add(10000)
	mu := sync.Mutex{}
	count := 0
	for i := 0; i < 10000; i++ {
		pool.AddTask(func() error {
			mu.Lock()
			defer mu.Unlock()
			count++
			wg.Done()
			return nil
		})
	}
	wg.Wait()
}

func BenchmarkTask(b *testing.B) {
	b.Run("NoWorkerPool", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			noPool()
		}
	})
	b.Run("UseWorkerPool", func(b *testing.B) {
		b.ReportAllocs()
		onErr := func(err error) {
			fmt.Println(err)
		}
		pool := NewWorkerPool(80, 160, time.Minute, onErr)
		for i := 0; i < b.N; i++ {
			useWorkerPool(pool)
		}
	})
}

func noPool() {
	var wg sync.WaitGroup
	wg.Add(100000)
	var count int64
	for i := 0; i < 100000; i++ {
		go func() {
			atomic.AddInt64(&count, 1)
			wg.Done()
		}()
	}
	wg.Wait()
}

func useWorkerPool(pool *WorkerPool) {
	var wg sync.WaitGroup
	wg.Add(100000)
	var count int64
	for i := 0; i < 100000; i++ {
		pool.AddTask(func() error {
			atomic.AddInt64(&count, 1)
			wg.Done()
			return nil
		})
	}
	wg.Wait()
}
