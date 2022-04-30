package ddio

import (
	"errors"
	"net"
	"strconv"
	"strings"
	"unsafe"
)

func noescape(pointer unsafe.Pointer) unsafe.Pointer {
	x := uintptr(pointer)
	return unsafe.Pointer(x ^ 0)
}

// 方便的双倍扩容函数
func doubleGrow(memPool *MemoryPool,oldBuf []byte) (newBuf []byte,bl bool) {
	bl = memPool.Grow(&oldBuf,(cap(oldBuf) / memPool.BlockSize()) * 2)
	if bl {
		newBuf = oldBuf
	}
	return
}

func parseAddress(addr string) (config NetPollConfig, argMap map[string]string, err error) {
	argSlice := strings.Split(strings.SplitN(addr, "?", 2)[1], "&")
	argMap = make(map[string]string, len(argSlice)/2)
	for _, v := range argSlice {
		kAndV := strings.Split(v, "=")
		argMap[kAndV[0]] = kAndV[1]
	}
	connProtocol := strings.Split(strings.SplitN(addr, "?", 2)[0], "//")
	switch {
	case strings.EqualFold(connProtocol[0], "tcp:"):
		ipSplit := strings.Split(connProtocol[1], ":")
		switch {
		case len(ipSplit) == 2:
			config.Protocol = TCP_V4
		case len(ipSplit) > 2:
			config.Protocol = TCP_V6
		default:
			err = errors.New("ip format not supported")
			return
		}
		// IPV6地址有简便表示多个零的写法，要对这种方法做特殊处理
		// 比如这个地址:fe80::1029:f994:b74a:7bef,::处缺少了3组零位
		// net.ParseIP中排除Port
		ip := net.ParseIP(connProtocol[1][:len(connProtocol[1]) - len(ipSplit[len(ipSplit) - 1]) - 1])
		config.IP = ip
		config.Port, err = strconv.Atoi(ipSplit[len(ipSplit)-1])
	default:
		return config, nil, errors.New("not supported protocol")
	}
	return
}
