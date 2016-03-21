package ip

import (
	//	"bytes"
	//	"encoding/binary"
	"fmt"
	//"log"
	"tcpip/net/ethernet"
	"tcpip/net/skb"
	"tcpip/util"
	//"net"
	//  "io"
	//	"strconv"
	//	"strings"
	//	"syscall"
	//	"time"
)

type IPHeader struct {
	VersionIhl uint8
	Tos        uint8
	TotLength  uint16
	Ident      uint16
	FragOff    uint16
	TTL        byte
	Protocol   byte
	Check      uint16 // byte 10
	Saddr      uint32 // byte 12
	Daddr      uint32 // byte 16

}

type IPPacket struct {
	Header *IPHeader
	Packet *[]byte
}

/** struct ip_options - IP Options
 *
 * @faddr - Saved first hop address
 * @is_data - Options in __data, rather than skb
 * @is_strictroute - Strict source route
 * @srr_is_hit - Packet destination addr was our one
 * @is_changed - IP checksum more not valid
 * @rr_needaddr - Need to record addr of outgoing dev
 * @ts_needtime - Need to record timestamp
 * @ts_needaddr - Need to record addr of outgoing dev
 */
type IPOptions struct {
	faddr          uint32
	optlen         uint8
	srr            uint8
	rr             uint8
	ts             uint8
	is_strictroute bool
	srr_is_hit     bool
	is_changed     bool
	rr_needaddr    bool
	ts_needtime    bool
	ts_needaddr    bool
	router_alert   uint8
	cipso          uint8
	__pad2         uint8
	__data         []byte
}

/*InetSkbParm flag*/
const IPSKB_FORWARDED = 1
const IPSKB_XFRM_TUNNEL_SIZE = 2
const IPSKB_XFRM_TRANSFORMED = 4
const IPSKB_FRAG_COMPLETE = 8
const IPSKB_REROUTED = 16

type InetSkbParm struct {
	opt   IPOptions /* Compiled IP options      */
	flags uint8
}

type IPHandle struct {
	EthHandle *ethernet.EthHandle
}

const LEN_IPHDR = 20

func (hdr *IPHeader) GetVersion() byte {
	return byte(hdr.VersionIhl >> 4)
}
func (hdr *IPHeader) SetVersion(ver uint8) *IPHeader {
	hdr.VersionIhl |= (0xf0 & (ver << 4))
	return hdr
}

// Bytes
func (hdr *IPHeader) SetHeaderLen(len uint8) *IPHeader {
	//hdr.VersionIhl |= (0x0F & ((len >> 2) << 8))
	hdr.VersionIhl |= (0x0F & (len >> 2))
	return hdr
}

// Bytes
func (hdr *IPHeader) GetHeaderLen() byte {
	return 4 * byte(0x0F&hdr.VersionIhl)
}
func (hdr *IPHeader) GetTotLength() uint16 {
	return ((hdr.TotLength >> 8) & 0xff) | ((hdr.TotLength << 8) & 0xff00)
}

func (hdr *IPHeader) SetDSCP(dscp byte) *IPHeader {
	hdr.Tos |= uint8(0x3F & (dscp << 2))
	return hdr
}

func (hdr *IPHeader) SetECN(ecn byte) *IPHeader {
	hdr.Tos |= uint8(0x3 & (ecn))
	return hdr
}

func (hdr *IPHeader) SetNoFrag() *IPHeader {
	//	hdr.FragOff = 0x0FFF & hdr.FragOff
	//	hdr.FragOff |= (0x2 << 13)
	hdr.FragOff = 0xFF0F & hdr.FragOff
	hdr.FragOff |= (0x2 << 5)

	fmt.Printf("hdr.FragOff:%d\n", hdr.FragOff)
	return hdr
}

func (ipHdr *IPHeader) String() string {
	if ipHdr == nil {
		return "<nil>"
	}
	return fmt.Sprintf("Version Data: %d \nLen (bytes): %d\nDSCP: %d\nECN: %d\nIdent: %d\nFlags: %d\nTTL: %d\nProto: %d\nSource IP: %s\nDestination IP: %s\nIP Proto: %d\nP Data Length: %d\n",
		ipHdr.GetVersion(), ipHdr.GetHeaderLen(), (0x3F & (ipHdr.Tos >> 2)), (0x03 & (ipHdr.Tos)),
		ipHdr.Ident, 0x7&(ipHdr.FragOff>>13), ipHdr.TTL, ipHdr.Protocol, ipHdr.Source(),
		ipHdr.Destination(), ipHdr.Protocol, ipHdr.GetTotLength())
}

func DotIP(addr uint32) string {
	//return fmt.Sprintf("%d.%d.%d.%d", addr[0], addr[1], addr[2], addr[3])
	return fmt.Sprintf("%d.%d.%d.%d", (addr & 0xff), (addr>>8)&0xff, (addr>>16)&0xff, (addr>>24)&0xff)
}

func (ipHdr *IPHeader) Source() string {
	return DotIP(ipHdr.Saddr)
}

func (ipHdr *IPHeader) Destination() string {
	return DotIP(ipHdr.Daddr)
}

/*
func (ipHdr *IPHeader) GetSaddr() uint32 {
	return (uint32)((24 << ipHdr.Saddr[0]) | (16 << ipHdr.Saddr[1]) | (8 << ipHdr.Saddr[2]) | ipHdr.Saddr[3])
}

func (ipHdr *IPHeader) GetDaddr() uint32 {
	return (uint32)((24 << ipHdr.Daddr[0]) | (18 << ipHdr.Daddr[1]) | (8 << ipHdr.Daddr[2]) | ipHdr.Daddr[3])
}
*/
func Start(device string, filter string) chan *skb.SkBuff {
	outQueue := make(chan *skb.SkBuff, 2048)
	//iphandle := new(IPHandle)
	//var err error
	inQueue := ethernet.Start(device, filter)
	//fmt.Printf("len=%d\n", len(c))
	go receive(inQueue, outQueue)
	return outQueue
}
func receive(in chan *skb.SkBuff, out chan *skb.SkBuff) {
	for {
		skbuf := <-in
		skbuf.NetworkHdr = util.Byte2Pointer(skbuf.Buffer[skbuf.NetworkIndex:])
		ipHdr := (*IPHeader)(skbuf.NetworkHdr)
		fmt.Println(ipHdr)
		skbuf.PayloadLen = util.Ntohs(ipHdr.TotLength) - 4*uint16(0x0F&ipHdr.VersionIhl)
		skbuf.TransportIndex = 4*uint16(0x0F&ipHdr.VersionIhl) + skbuf.NetworkIndex
		fmt.Println(ipHdr)
		out <- skbuf
	}
}
