package http_parser

import (
	"bufio"
	"fmt"
	"net"
	"testing"
)

func TestHttpPaser(t *testing.T) {

	paserobj := NewHttpPaser()

	a, err := net.Listen("tcp", "127.0.0.1:8088")
	if err != nil {
		panic(err)
	}
	for {
		cl, _ := a.Accept()
		fmt.Print(cl.LocalAddr())
		go paserobj.PaserHttp(*bufio.NewReader(cl), func(req Request) {

		})
	}

}
