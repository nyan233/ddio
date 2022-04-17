package main

import (
	"fmt"
	"github.com/zbh255/nyan/event_poll"
	"net"
	"time"
)

type SimpleHttpEchoServer struct {

}

func (s *SimpleHttpEchoServer) OnData(conn ddio.Conn) error {
	conn.WriteBytes([]byte("HTTP/1.1 200 OK\r\nServer: ddio\r\nContent-Type: text/plain\r\nDate: "))
	conn.WriteBytes(time.Now().AppendFormat([]byte{}, "Mon, 02 Jan 2006 15:04:05 GMT"))
	conn.WriteBytes([]byte("\r\nContent-Length: 12\r\n\r\nHello World!"))
	return nil
}

func (s *SimpleHttpEchoServer) OnClose(ev ddio.Event) error {
	fmt.Println("connection closed")
	return nil
}

func (s *SimpleHttpEchoServer) OnError(ev ddio.Event, err error) {
	fmt.Println("connection error: ", err)
}

func main() {
	config := &ddio.EngineConfig{
		ConnHandler:    &SimpleHttpEchoServer{},
		NBalance: func() ddio.Balanced {
			return &ddio.RoundBalanced{}
		},
		NetPollConfig: &ddio.NetPollConfig{
			Protocol: 0x1,
			IP: net.ParseIP("192.168.1.150"),
			Port: 8080,
		},
	}
	engine,err := ddio.NewEngine(ddio.NewTCPListener(ddio.EVENT_LISTENER),config)
	if err != nil {
		panic(err)
	}
	engine.Run()
	_ = engine.Close()
}