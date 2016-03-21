package ethernet

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/google/gopacket"
	//	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"log"
	"slab"
	"tcpip/net/skb"
	"tcpip/util"
	"time"
	//	"unsafe"
)

const LEN_ETHHDR = 14

type EthHeader struct {
	DstMAC   [6]byte
	SrcMAC   [6]byte
	Protocol uint16
}

type EthPacket struct {
	Header EthHeader
	Packet []byte
}

type EthHandle struct {
	handle *pcap.Handle
	//	packetSource *gopacket.PacketSource
}

var pool *slab.SyncPool

func (hdr *EthHeader) SetMac(src, dst [6]byte) *EthHeader {
	hdr.SrcMAC, hdr.DstMAC = src, dst
	return hdr
}

func (hdr *EthHeader) String() string {
	return fmt.Sprintf("Src MAC %x:%x:%x:%x:%x:%x Dst MAC %x:%x:%x:%x:%x:%x protocol %x",
		hdr.SrcMAC[0], hdr.SrcMAC[1], hdr.SrcMAC[2], hdr.SrcMAC[3], hdr.SrcMAC[4], hdr.SrcMAC[5],
		hdr.DstMAC[0], hdr.DstMAC[1], hdr.DstMAC[2], hdr.DstMAC[3], hdr.DstMAC[4], hdr.DstMAC[5], hdr.Protocol)
}

var (
	//	device       string = "eth0"
	snapshot_len int32 = 1024
	promiscuous  bool  = true
	err          error
	//	filter       string        = "udp and port 65001"
	timeout      time.Duration = -1 //30 * time.Second
	handle       *pcap.Handle
	packetSource *gopacket.PacketSource
	ethHandle    *EthHandle
)

/*
type SKBQueue struct {
	C chan *skb.SkBuff
}
*/
func Start(device string, filter string) chan *skb.SkBuff /*SKBQueue*/ {
	skbChannel := make(chan *skb.SkBuff, 2048)
	// Open device
	handle, err = pcap.OpenLive(device, snapshot_len, promiscuous, timeout)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%v, handle:%v\n\n", err, handle)

	err = handle.SetBPFFilter(filter)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Only capturing" + filter + "packets.")

	pool = slab.NewSyncPool(
		1500,         // The smallest chunk size is 64B.
		1500*1024*20, // The largest chunk size is 64KB.
		2,            // Power of 2 growth in chunk size.
	)

	//pool.Free(buf)
	ethHandle = &EthHandle{handle}
	fmt.Printf("----handle:%v\n", handle)
	//c := make(chan *EthHandle, 1)
	//go test()
	//	queue := &SKBQueue{skbChannel}
	//go receive(queue.C)
	go receive(skbChannel)
	//go send(ethHandle)
	//	return queue
	return skbChannel
}

const DATA_OFFSET = 0 //(14 + 28)

func receive(c chan *skb.SkBuff) {

	//ethHandle := <-handle
	fmt.Printf("Ethernet=>Reading with %v\n", ethHandle.handle)
	packetSource = gopacket.NewPacketSource(ethHandle.handle, ethHandle.handle.LinkType())

	defer ethHandle.handle.Close()

	packetSource := gopacket.NewPacketSource(ethHandle.handle, ethHandle.handle.LinkType())
	for packet := range packetSource.Packets() {
		//		printPacketInfo(packet)
		pktLen := len(packet.Data())
		skbuf := new(skb.SkBuff)
		buf := pool.Alloc(pktLen - DATA_OFFSET)
		reader := bytes.NewReader(packet.Data()[DATA_OFFSET:pktLen])
		/*
			reader := bytes.NewReader(packet.Data()[LEN_ETHHDR+28 : 28+LEN_ETHHDR])
		*/
		err = binary.Read(reader, binary.BigEndian, &buf)
		if err != nil {
			log.Fatal(err)
		}

		//skbuf.Data = (*[]byte)(util.Byte2Pointer(&buf))
		skbuf.Buffer = buf
		//	skbuf.Data = &buf
		fmt.Printf("packet len =%d, data:%p,%v\n", pktLen, buf, buf)

		skbuf.MacIndex = 0
		skbuf.NetworkIndex = LEN_ETHHDR
		skbuf.MacHdr = util.Byte2Pointer(skbuf.Buffer[:LEN_ETHHDR])
		c <- skbuf
		fmt.Printf("recv:%s\n", (*EthHeader)(skbuf.MacHdr))
	}
	//return skbuf, err
}

//func (ethHandle *EthHandle) send(data []byte) (int, error) {
func send(skbuf *skb.SkBuff) (int, error) {
	fmt.Println("Ethernet start send packet:%v", skbuf.Buffer)
	//	hdr := (*EthHeader)(skbuf.Data)
	macHdr := (*EthHeader)(skbuf.MacHdr)
	fmt.Printf("send:%s\n", macHdr)
	err = ethHandle.handle.WritePacketData(skbuf.Buffer)
	if err != nil {
		log.Fatal(err)
	}
	return len(skbuf.Buffer), err
}
