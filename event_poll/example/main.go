package main

import (
	"github.com/zbh255/bilog"
	"github.com/zbh255/nyan/event_poll"
	"net"
	"os"
)

type SimpleHttpEchoServer struct {

}

func (s *SimpleHttpEchoServer) OnRead(readBuffer, writeBuffer []byte, flags ddio.EventFlags) error {
	copy(readBuffer,writeBuffer)
	return nil
}

func (s *SimpleHttpEchoServer) OnWrite(writeBuffer []byte, flags ddio.EventFlags) error {
	panic("implement me")
}

func (s *SimpleHttpEchoServer) OnClose(ev ddio.Event) error {
	panic("implement me")
}

func (s *SimpleHttpEchoServer) OnError(ev ddio.Event, err error) {
	panic("implement me")
}

func main() {
	logger := bilog.NewLogger(os.Stdout,bilog.PANIC,bilog.WithTimes(),bilog.WithCaller())
	var onErr ddio.ErrorHandler = func(err error) {
		logger.ErrorFromErr(err)
	}

	config := &ddio.DisPatcherConfig{
		ConnEvent:      ddio.EVENT_READ,
		ConnHandler:    &SimpleHttpEchoServer{},
		ConnErrHandler: onErr,
		EngineConfig: &ddio.NetPollConfig{
			Protocol: 0x1,
			IP: net.ParseIP("0.0.0.0"),
			Port: 8080,
		},
	}
	engine,err := ddio.NewEngine(ddio.NewTCPListener(ddio.EPOLLIN),onErr,config)
	if err != nil {
		logger.PanicFromErr(err)
	}
	engine.Run()
}