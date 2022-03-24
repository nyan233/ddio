package http_parser

import (
	"bufio"
	"fmt"
	"io"
	"sync"
)

type httpPaser struct {
	//减少gc 复用对象
	HttpContial sync.Pool
}

func NewHttpPaser() IHttpPaser {
	obj := httpPaser{}

	obj.HttpContial = sync.Pool{
		New: func() interface{} { return Request{} },
	}

	return obj
}

//序列化http协议
func (H httpPaser) PaserHttp(read bufio.Reader, funcs PaserHttpFunc) {

	req := H.HttpContial.Get().(Request)

	for {
		body, err := read.ReadString('\n')
		if err == io.EOF {
			fmt.Println("链接被关闭")
			break
		}
		fmt.Println(body)
	}
	funcs(req)

}

//解析http协议
func HttpAnalysis(msg []byte) {

}
