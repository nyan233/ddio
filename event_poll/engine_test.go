package ddio

import (
	"errors"
	"fmt"
	"golang.org/x/sys/unix"
	"math/rand"
	"net"
	"sync"
	"testing"
	"time"
)

type testEchoServer struct {

}

func (t *testEchoServer) OnInit() ConnConfig {
	return DefaultConfig
}

func (t *testEchoServer) OnData(conn *TCPConn) error {
	buffer := make([]byte, 0, 256)
	buffer = append(buffer, "HTTP/1.1 200 OK\r\nServer: ddio\r\nContent-Type: text/plain\r\nDate: "...)
	buffer = append(buffer, time.Now().AppendFormat([]byte{}, "Mon, 02 Jan 2006 15:04:05 GMT")...)
	buffer = append(buffer, "\r\nContent-Length: 12\r\n\r\nHello World!"...)
	conn.WriteBytes(buffer)
	return nil
}

func (t *testEchoServer) OnClose(ev Event) error {
	return unix.Close(int(ev.fd()))
}

func (t *testEchoServer) OnError(ev Event, err error) {
	unix.Close(int(ev.fd()))
}

func TestEngine(t *testing.T) {
	addrs := []string{
		"0.0.0.0:4076",
		"0.0.0.0:4077",
		"0.0.0.0:4078",
		"0.0.0.0:4079",
	}
	// 测试每个地址的客户端数量
	addrClients := 10
	// 保存客户端连接的map
	client := make(map[string][]net.Conn,addrClients * len(addrs))
	config := &EngineConfig{
		ConnHandler: &testEchoServer{},
		NBalance: func() Balanced {
			return &RoundBalanced{}
		},
		MultiAddr: func() []string{
			tmp := make([]string,len(addrs))
			for k,v := range addrs {
				tmp[k] = fmt.Sprintf("tcp://%s?level=%d",v,10 / len(addrs))
			}
			return tmp
		}(),
	}
	_, err := NewEngine(NewTCPListener(EVENT_LISTENER),config)
	if err != nil {
		t.Fatal(err)
	}
	for _,v := range addrs {
		conns := make([]net.Conn,addrClients)
		for i := 0; i < addrClients; i++ {
			conn, err := net.Dial("tcp", v)
			if err != nil {
				t.Fatal(err)
			}
			conns[i] = conn
		}
		client[v] = conns
	}
	// 4096-32768Byte data
	initV := 12
	for i := 0; i < 4; i++ {
		runStr := fmt.Sprintf("%dB-Send",1 << (initV + i))
		t.Run(runStr, func(t *testing.T) {
			var wg sync.WaitGroup
			wg.Add(len(client) * addrClients)
			for k := range client {
				sendBuf := randomNBytes(1 << (initV + i))
				for i := range client[k] {
					conn := client[k][i]
					go func() {
						defer func() {
							wg.Done()
							err := recover()
							if err != nil {
								t.Error(err)
							}
						}()
						rBuffer := [512]byte{}
						writeN, err2 := conn.Write(sendBuf)
						if err2 != nil {
							panic(err2)
						}
						if writeN != len(sendBuf) {
							panic(errors.New("write bytes not equal"))
						}
						_, err := conn.Read(rBuffer[:])
						if err != nil {
							panic(err)
						}
						return
					}()
				}
			}
			wg.Wait()
		})
	}
	//server.Close()
}

func randomNBytes(n int) []byte {
	rand.Seed(time.Now().UnixNano())
	tmp := make([]byte,n)
	for k := range tmp{
		tmp[k] = byte(rand.Intn(26) + 65)
	}
	return tmp
}