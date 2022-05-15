package internal

import (
	"fmt"
	"net"
	"runtime"
	"sync"
	"testing"
	"time"
)

const (
	sleepTime = time.Nanosecond * 10
	nTask = 1000000
)

func BenchmarkTask(b *testing.B) {
	b.Run("NoWorkerPool", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			noPool()
		}
	})
	b.Run("UseWorkerPool", func(b *testing.B) {
		b.ReportAllocs()
		var wg sync.WaitGroup
		onErr := func(err error) {
			fmt.Println(err)
		}
		handle := func(data interface{}) error{
			time.Sleep(sleepTime)
			wg.Done()
			return nil
		}
		pool := NewWorkerPool(64, 256 * runtime.NumCPU(),handle,onErr)
		for i := 0; i < b.N; i++ {
			wg.Add(nTask)
			for i := 0; i < nTask; i++ {
				pool.PushTask(0)
			}
			wg.Wait()
		}
	})
}

func noPool() {
	var wg sync.WaitGroup
	wg.Add(nTask)
	for i := 0; i < nTask; i++ {
		go func() {
			time.Sleep(sleepTime)
			wg.Done()
		}()
	}
	wg.Wait()
}

func BenchmarkSocketClose(b *testing.B) {
	server,err := net.Listen("tcp","127.0.0.1:4567")
	if err != nil {
		b.Fatal(err)
	}
	go func() {
		for {
			conn, _ := server.Accept()
			_ = conn.Close()
		}
	}()
	nConn := 5000
	conns := make([]net.Conn,0,nConn)
	for i := 0; i < nConn; i++ {
		conn, err := net.Dial("tcp", "127.0.0.1:4567")
		if err != nil {
			b.Fatal(err)
		}
		conns = append(conns,conn)
	}
	b.Run("NoPoolRead", func(b *testing.B) {
		b.ReportAllocs()
		var wg sync.WaitGroup
		for i := 0; i < b.N; i++ {
			wg.Add(nConn)
			for i := 0 ; i < nConn; i++{
				go func(conn net.Conn) {
					_ = conn.Close()
					wg.Done()
				}(conns[i])
			}
			wg.Wait()
		}
	})
	b.Run("UsePoolRead", func(b *testing.B) {
		b.ReportAllocs()
		var wg sync.WaitGroup
		handleFn := func(data interface{}) error {
			conn := data.(net.Conn)
			_ = conn.Close()
			wg.Done()
			return nil
		}
		onErr := func(err2 error) {
			return
		}
		pool := NewWorkerPool(64,1024,handleFn,onErr)
		for i := 0; i < b.N; i++ {
			wg.Add(nConn)
			for i := 0; i < nConn; i++ {
				pool.PushTask(conns[i])
			}
			wg.Wait()
		}
	})
}
