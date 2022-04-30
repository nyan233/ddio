package main

import (
	"fmt"
	"github.com/zbh255/bilog"
	ddio "github.com/zbh255/nyan/event_poll"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"
)

type SimpleHttpEchoServer struct {
}

func (s *SimpleHttpEchoServer) OnInit() ddio.ConnConfig {
	return ddio.ConnConfig{OnDataNBlock: 1}
}

func (s *SimpleHttpEchoServer) OnData(conn *ddio.TCPConn) error {
	buffer := make([]byte, 0, 256)
	buffer = append(buffer, "HTTP/1.1 200 OK\r\nServer: ddio\r\nContent-Type: text/plain\r\nDate: "...)
	buffer = append(buffer, time.Now().AppendFormat([]byte{}, "Mon, 02 Jan 2006 15:04:05 GMT")...)
	buffer = append(buffer, "\r\nContent-Length: 12\r\n\r\nHello World!"...)
	conn.WriteBytes(buffer)
	return nil
}

func (s *SimpleHttpEchoServer) OnClose(ev ddio.Event) error {
	fmt.Println("connection closed")
	return nil
}

func (s *SimpleHttpEchoServer) OnError(ev ddio.Event, err error) {
	fmt.Println("connection error: ", err)
}

var logger = bilog.NewLogger(os.Stdout, bilog.DEBUG, bilog.WithTimes(), bilog.WithCaller(), bilog.WithTopBuffer(2))

func main() {
	go func() {
		logger.Debug(http.ListenAndServe("0.0.0.0:9090", nil).Error())
	}()
	config := &ddio.EngineConfig{
		ConnHandler: &SimpleHttpEchoServer{},
		NBalance: func() ddio.Balanced {
			return &ddio.RoundBalanced{}
		},
		MultiAddr: []string{
			"tcp://0.0.0.0:8080?level=10",
		},
	}
	_, err := ddio.NewEngine(ddio.NewTCPListener(ddio.EVENT_LISTENER), config)
	if err != nil {
		panic(err)
	}

	select {}
}
