package ip

import (
	//	"bufio"
	//	"bytes"
	//	"encoding/binary"
	"fmt"
	"log"
	"tcpip/net/ethernet"
	"tcpip/net/skb"
	//	"tcpip/net/tcp"
	"tcpip/util"
)

type IPConn struct {
	RemoteIP uint32
	LocalIP  uint32
	EthConn  *ethernet.EthConn
	ipHeader *IPHeader
	//ipHandle *IPHandle
}

func (ipConn *IPConn) Send(skbuf *skb.SkBuff) (int, error) {

	b := skbuf.Buffer[skbuf.NetworkIndex : skbuf.NetworkIndex+LEN_IPHDR]
	ipHeader := (*IPHeader)(util.Byte2Pointer(b))
	ipHeader.TotLength = util.Htons(uint16(skbuf.PayloadLen + LEN_IPHDR))

	ipHeader.Saddr = ipConn.ipHeader.Saddr
	ipHeader.Daddr = ipConn.ipHeader.Daddr
	ipHeader.Protocol = ipConn.ipHeader.Protocol

	ipHeader.VersionIhl = ipConn.ipHeader.VersionIhl
	ipHeader.Tos = ipConn.ipHeader.Tos
	ipHeader.Ident = ipConn.ipHeader.Ident
	ipHeader.FragOff = ipConn.ipHeader.FragOff
	ipHeader.TTL = ipConn.ipHeader.TTL
	ipHeader.Check = ipConn.ipHeader.Check

	//fmt.Printf("send ip:%d-->%d\n", ipHeader.Saddr, ipHeader.Daddr)
	ipHeader.Check = util.Ntohs(util.Checksum(b))
	*(*uint16)(skbuf.CheckPotioner) = util.Htons(util.Checksum(skbuf.Buffer[skbuf.NetworkIndex:skbuf.DataEnd]))
	skbuf.MacIndex = 0
	//tcp checksum
	//a := skbuf.Buffer[skbuf.NetworkIndex:skbuf.DataEnd]
	//fmt.Printf("tcp checksum:%v\n", util.Checksum(skbuf.Buffer[skbuf.NetworkIndex:skbuf.DataEnd]))
	fmt.Printf("-----send----\n:%s\n", ipHeader)
	verified := util.VerifyChecksum(b)
	if !verified {
		log.Fatal("Unable to verify IP Checksum")
	}
	c := skbuf.Buffer[skbuf.NetworkIndex:skbuf.DataEnd]
	verified = util.VerifyChecksum(c)
	if !verified {
		log.Fatal("Unable to verify TCP Checksum")
	}
	//fmt.Printf("ip check:%x,tcp check:%x\n", ipHeader.Check, *(*uint16)(skbuf.CheckPotioner))

	return ipConn.EthConn.Send(skbuf)
	/*
		ipHeader := (*IPHeader)(skbuf.Data)

		ipHeader.TotLength = uint16(skbuf.PayloadLen + LEN_IPHDR)

		ipHeader.VersionIhl = ipConn.ipHeader.VersionIhl
		ipHeader.Tos = ipConn.ipHeader.Tos
		ipHeader.Ident = ipConn.ipHeader.Ident
		ipHeader.FragOff = ipConn.ipHeader.FragOff
		ipHeader.TTL = ipConn.ipHeader.TTL
		ipHeader.Protocol = ipConn.ipHeader.Protocol
		ipHeader.Check = ipConn.ipHeader.Check

		ipHeader.Saddr = ipConn.ipHeader.Saddr
		ipHeader.Daddr = ipConn.ipHeader.Daddr
		var hdrBuf bytes.Buffer
		hw := bufio.NewWriter(&hdrBuf)
		binary.Write(hw, binary.BigEndian, ipHeader)
		hw.Flush()
		ipHeader.Check = util.Checksum(hdrBuf.Bytes())
		fmt.Printf("IPConn=>Packet is %v\n", ipHeader)
		skbuf.Data = unsafe.Pointer(uintptr(skbuf.Data) - uintptr(14))
		return ipConn.ipHandle.ethHandle.Send(skbuf)
	*/
}

//func NewIPConn(localIP uint32, remoteIP uint32, ipHandle *IPHandle, skbuf *skb.SkBuff) (ipConn *IPConn, err error) {
func NewIPConn(localIP uint32, remoteIP uint32, skbuf *skb.SkBuff) (ipConn *IPConn, err error) {

	ipConn = &IPConn{RemoteIP: remoteIP,
		LocalIP: localIP}
	ethHdr := (*ethernet.EthHeader)(skbuf.MacHdr)

	//	ethheader *EthHeader, ethHandle *EthHandle
	ipConn.EthConn, _ = ethernet.NewEthernetConn(ethHdr)
	var hdr IPHeader
	hdr.SetVersion(4)
	hdr.SetHeaderLen(20)
	hdr.SetDSCP(0)
	hdr.SetECN(0)
	hdr.SetNoFrag()
	hdr.TTL = 64
	hdr.Protocol = 6
	hdr.Saddr = localIP
	hdr.Daddr = remoteIP
	//hdr.Ident = 1234
	ipConn.ipHeader = &hdr
	err = nil
	return
}
