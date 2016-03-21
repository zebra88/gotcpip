package socket

import (
	"tcpip/util"
)

type socket struct {
	Proto      *ZBProtocol
	LocalAddr  uint32
	RemoteAddr uint32
	LocalPort  uint16
	RemotePort uint16
}

func OpenSocket(net uint16, proto uint16) {

}
