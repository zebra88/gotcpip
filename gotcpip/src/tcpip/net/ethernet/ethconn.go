package ethernet

import (
	//	"bufio"
	//	"bytes"
	//	"encoding/binary"
	//	"fmt"
	//	"log"
	"tcpip/net/skb"
	"tcpip/util"
)

type EthConn struct {
	Header    EthHeader
	EthHandle *EthHandle
}

func (ethconn *EthConn) Send(skbuf *skb.SkBuff) (int, error) {
	/*
		skbuf.Buffer[0], skbuf.Buffer[1], skbuf.Buffer[2], skbuf.Buffer[3], skbuf.Buffer[4], skbuf.Buffer[5] =

			ethconn.Header.DstMAC[0], ethconn.Header.DstMAC[1],
			ethconn.Header.DstMAC[2], ethconn.Header.DstMAC[3],
			ethconn.Header.DstMAC[4], ethconn.Header.DstMAC[5]

		skbuf.Buffer[6], skbuf.Buffer[7], skbuf.Buffer[8], skbuf.Buffer[9], skbuf.Buffer[10], skbuf.Buffer[11] =
			ethconn.Header.SrcMAC[0], ethconn.Header.SrcMAC[1],
			ethconn.Header.SrcMAC[2], ethconn.Header.SrcMAC[3],
			ethconn.Header.SrcMAC[4], ethconn.Header.SrcMAC[5]
	*/
	skbuf.Buffer[0], skbuf.Buffer[1], skbuf.Buffer[2], skbuf.Buffer[3], skbuf.Buffer[4], skbuf.Buffer[5] =
		ethconn.Header.SrcMAC[0], ethconn.Header.SrcMAC[1],
		ethconn.Header.SrcMAC[2], ethconn.Header.SrcMAC[3],
		ethconn.Header.SrcMAC[4], ethconn.Header.SrcMAC[5]

	skbuf.Buffer[6], skbuf.Buffer[7], skbuf.Buffer[8], skbuf.Buffer[9], skbuf.Buffer[10], skbuf.Buffer[11] =

		ethconn.Header.DstMAC[0], ethconn.Header.DstMAC[1],
		ethconn.Header.DstMAC[2], ethconn.Header.DstMAC[3],
		ethconn.Header.DstMAC[4], ethconn.Header.DstMAC[5]
	skbuf.Buffer[12] = 8 //ethconn.Header.Protocol
	skbuf.MacHdr = util.Byte2Pointer(skbuf.Buffer[0:14])
	//return ethconn.EthHandle.send(skbuf)
	return send(skbuf)
}

func NewEthernetConn(ethheader *EthHeader) (ethconn *EthConn, err error) {
	//ethconn = &EthConn{EthHandle: ethHandle}
	ethconn = &EthConn{}
	ethconn.Header.SrcMAC, ethconn.Header.DstMAC, ethconn.Header.Protocol =
		ethheader.SrcMAC, ethheader.DstMAC, ethheader.Protocol

	err = nil
	return
}
