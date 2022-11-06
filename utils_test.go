package ddio

import (
	"net"
	"reflect"
	"strconv"
	"testing"
)

func TestProtocolParse(t *testing.T) {
	t.Run("IP_V4", func(t *testing.T) {
		addr := "tcp://127.0.0.1:8080?level=2"
		cmpAddr := NetPollConfig{
			Protocol: TCP_V4,
			IP:       net.ParseIP("127.0.0.1"),
			Port:     8080,
		}
		testFn(t, addr, cmpAddr, 2)
	})
	t.Run("IP_V6", func(t *testing.T) {
		addr := "tcp://fe80::1029:f994:b74a:7bef:8099?level=10"
		cmpAddr := NetPollConfig{
			Protocol: TCP_V6,
			IP:       net.ParseIP("fe80::1029:f994:b74a:7bef"),
			Port:     8099,
		}
		testFn(t, addr, cmpAddr, 10)
	})
}

func testFn(t *testing.T, addr string, cmpAddr NetPollConfig, cmpLevel int) {
	parseAddr, argMap, err := parseAddress(addr)
	if err != nil {
		t.Error(err)
		return
	}
	if level, _ := strconv.Atoi(argMap["level"]); level != cmpLevel {
		t.Error("level not equal")
		return
	}
	if !reflect.DeepEqual(parseAddr, cmpAddr) {
		t.Error("parse addr not equal")
		return
	}
}
