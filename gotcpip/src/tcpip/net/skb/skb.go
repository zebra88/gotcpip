package skb

import (
	//"tcpip/net/ethernet"
	//"tcpip/net/ip"
	//"tcpip/net/tcp"
	"unsafe"
)

type SKBOption struct {
	OptionLen uint8
	cb        [40]byte
}
type SkBuff struct {
	TransportHdr  unsafe.Pointer
	NetworkHdr    unsafe.Pointer
	MacHdr        unsafe.Pointer
	CheckPotioner unsafe.Pointer
	Size          uint16
	Data          unsafe.Pointer
	PayloadLen    uint16
	//Option         SKBOption
	CB             [40]byte
	AppIndex       uint16
	TransportIndex uint16
	NetworkIndex   uint16
	MacIndex       uint16
	DataEnd        uint16
	Buffer         []byte
}

func NewSkb() (skb *SkBuff, err error) {
	skb = new(SkBuff)
	if err != nil {
		return nil, err
	}

	return skb, nil
}
