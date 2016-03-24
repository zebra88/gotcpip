package stack

import (
	//"fmt"
	"rbtree"
	"tcpip/net/ip"
	"tcpip/net/protocol"
	"tcpip/net/tcp"
)

func InitStack() int {
	rbtree.InitRBtreeMemPool()
	protocol.ProtocolInit(ip.ProtoIpv4)
	protocol.ProtocolInit(tcp.ProtoTcp)

	return 0
}
