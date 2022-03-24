package http_parser

import (
	"bufio"
)

//解析后的回调
type PaserHttpFunc func(req Request)

type IHttpPaser interface {
	PaserHttp(read bufio.Reader, funcs PaserHttpFunc)
}
