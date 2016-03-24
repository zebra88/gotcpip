package protocol

import (
	"tcpip/net/skb"
)

const LOOP_DIR_IN = 1
const LOOP_DIR_OUT = 2

type ZBLayer uint8

type AllocFunc func(proto *ZBProtocol, size uint16) *skb.SkBuff
type ProcessFunc func(proto *ZBProtocol, _skb *skb.SkBuff) int

type ZBProtocol struct {
	Name        string
	Hash        uint32
	Layer       ZBLayer
	ProtoNumber uint16

	ChanIn  chan *skb.SkBuff
	ChanOut chan *skb.SkBuff

	QueueIn  *skb.SkBuffHead
	QueueOut *skb.SkBuffHead
	//	map[string]interface{}
	Alloc      AllocFunc
	Push       ProcessFunc
	ProcessOut ProcessFunc
	ProcessIn  ProcessFunc

	//    int (*push)();    /* Push function, for active outgoing pkts from above */
	//    uint16_t (*get_mtu)(protocol *self);
}

func IS_IPV6(f interface{}) bool {
	switch v := f.(type) {
	case *skb.SkBuff:

		return v != nil && v.Buffer[v.NetworkIndex]&0xf0 == 0x60
	default:
		return false
	}

}

func IS_IPV4(f interface{}) bool {
	switch v := f.(type) {
	case *skb.SkBuff:
		return v != nil && v.Buffer[v.NetworkIndex]&0xf0 == 0x40
	default:
		return false
	}
}

const MAX_PROTOCOL_NAME = 16
