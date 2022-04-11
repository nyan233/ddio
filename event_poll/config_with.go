package ddio

import "net"

type NetPollConfig struct {
	Protocol int
	IP   net.IP
	Port int
}
