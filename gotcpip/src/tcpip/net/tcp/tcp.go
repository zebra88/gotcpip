package tcp

// http://www.rfc-editor.org/rfc/rfc793.txt

import (
	//	"bytes"
	//	"encoding/binary"
	"fmt"
	"tcpip/net/ip"
	"tcpip/net/skb"
	"tcpip/util"
)

/*
 *  TCP option
 */
const TCPOPT_NOP = 1       /* Padding */
const TCPOPT_EOL = 0       /* End of options */
const TCPOPT_MSS = 2       /* Segment size negotiating */
const TCPOPT_WINDOW = 3    /* Window scaling */
const TCPOPT_SACK_PERM = 4 /* SACK Permitted */
const TCPOPT_SACK = 5      /* SACK Block */
const TCPOPT_TIMESTAMP = 8 /* Better RTT estimations/PAWS */
const TCPOPT_MD5SIG = 19   /* MD5 Signature (RFC2385) */
/*
 *     TCP option lengths
 */

const TCPOLEN_MSS = 4
const TCPOLEN_WINDOW = 3
const TCPOLEN_SACK_PERM = 2
const TCPOLEN_TIMESTAMP = 10
const TCPOLEN_MD5SIG = 18

/* But this is what stacks really send out. */
const TCPOLEN_TSTAMP_ALIGNED = 12
const TCPOLEN_WSCALE_ALIGNED = 4
const TCPOLEN_SACKPERM_ALIGNED = 4
const TCPOLEN_SACK_BASE = 2
const TCPOLEN_SACK_BASE_ALIGNED = 4
const TCPOLEN_SACK_PERBLOCK = 8
const TCPOLEN_MD5SIG_ALIGNED = 20
const TCPOLEN_MSS_ALIGNED = 4

/*
tcp flag
*/
const CTRL_URG = 1 << (5 + 8) // Urgent
const CTRL_ACK = 1 << (4 + 8)
const CTRL_PSH = 1 << (3 + 8)
const CTRL_RST = 1 << (2 + 8)
const CTRL_SYN = 1 << (1 + 8)
const CTRL_FIN = 1 << 8

const FOUR_BITS = 0xF

/* NOTE: These must match up to the flags byte in a
 *       real TCP header.
 */
const TCPCB_FLAG_FIN = 0x01
const TCPCB_FLAG_SYN = 0x02
const TCPCB_FLAG_RST = 0x04
const TCPCB_FLAG_PSH = 0x08
const TCPCB_FLAG_ACK = 0x10
const TCPCB_FLAG_URG = 0x20
const TCPCB_FLAG_ECE = 0x40
const TCPCB_FLAG_CWR = 0x80

const TCPCB_SACKED_ACKED = 0x01   /* SKB ACK'd by a SACK block    */
const TCPCB_SACKED_RETRANS = 0x02 /* SKB retransmitted        */
const TCPCB_LOST = 0x04           /* SKB is lost          */
const TCPCB_TAGBITS = 0x07        /* All tag bits         */

const TCPCB_EVER_RETRANS = 0x80 /* Ever retransmitted frame */
const TCPCB_RETRANS = (TCPCB_SACKED_RETRANS | TCPCB_EVER_RETRANS)

type tcp_skb_cb struct {
	header  ip.InetSkbParm /* For incoming frames      */
	seq     uint32         /* Starting sequence number */
	end_seq uint32         /* SEQ + FIN + SYN + datalen    */
	when    uint32         /* used to compute rtt's    */
	flags   uint8          /* TCP header flags.        */

	sacked  uint8  /* State flags for SACK/FACK.   */
	ack_seq uint32 /* Sequence number ACK'd    */

}
type TCPOptionsReceived struct {
	/*  PAWS/RTTM data  */
	ts_recent_stamp uint64 /* Time we stored ts_recent (for aging) */
	ts_recent       uint32 /* Time stamp to echo next      */
	rcv_tsval       uint32 /* Time stamp value                 */
	rcv_tsecr       uint32 /* Time stamp echo reply            */
	saw_tstamp      uint16 /* Saw TIMESTAMP on last packet     */

	tstamp_ok  bool  /* TIMESTAMP seen on SYN packet     */
	dsack      bool  /* D-SACK is scheduled          */
	wscale_ok  bool  /* Wscale seen on SYN packet        */
	sack_ok    uint8 /* SACK seen on SYN packet    4bit  */
	snd_wscale uint8 /* Window scaling received from sender 4bit */
	rcv_wscale uint8 /* Window scaling to send to receiver  4bit */
	/*  SACKs data  */
	num_sacks uint8  /* Number of SACK blocks        */
	user_mss  uint16 /* mss requested by user in ioctl */
	mss_clamp uint16 /* Maximal mss, negotiated at connection setup */
}

type TCPHeader struct {
	SourcePort uint16 // 0
	DestPort   uint16 // 2

	Seq uint32 // 4

	Ack uint32 // 8 Last seen SEQ + 1

	DataOffsetAndFlags uint16 // 12 Data offset = 4 (number of 32 bit words before data), reserved = 6, control bits = 6
	Window             uint16 // 14 how many octets we can handle

	Checksum      uint16 // 16 16-bit 1s complement sum of 16 bit words in header and text
	UrgentPointer uint16 // 18 If URG set, then this is the index to octet after urgent data
}

type TCPOption struct {
	opt [20]byte
}
type TCPPacket struct {
	Header            *TCPHeader
	OptionsAndPadding []byte // 20
	Data              []byte
}

/*
func (packet *TCPPacket) String() string {
	if packet == nil {
		return "<nil>"
	}
	return packet.Header.String() + fmt.Sprintf("Options=%v packet.Data=%d ", packet.OptionsAndPadding, len(packet.Data))
}

func (packet *TCPPacket) IsSYN() bool {
	return packet.Header.DataOffsetAndFlags&CTRL_SYN != 0
}

func (packet *TCPPacket) IsACK() bool {
	return packet.Header.DataOffsetAndFlags&CTRL_ACK != 0
}
*/

func (hdr *TCPHeader) IsSYN() bool {
	return hdr.DataOffsetAndFlags&CTRL_SYN != 0
}

func (hdr *TCPHeader) IsACK() bool {
	return hdr.DataOffsetAndFlags&CTRL_ACK != 0
}

// Acknowledge the up to the given sequence number
// This is not SACK
func (hdr *TCPHeader) ACK(seq uint32) {
	hdr.Ack = seq
	hdr.DataOffsetAndFlags |= CTRL_ACK
}

func (hdr *TCPHeader) String() string {
	return fmt.Sprintf("Source:%d Destination:%d SEQ:%d ACK:%d SYN? %t ACK? %t PSH? %t FIN? %t Window: %d %x Checksum: %d (%x) %b, Data Offset:%d, Options and Padding Size:%d",
		util.Ntohs(hdr.SourcePort), util.Ntohs(hdr.DestPort), util.Ntohl(hdr.Seq), util.Ntohl(hdr.Ack), hdr.DataOffsetAndFlags&CTRL_SYN != 0,
		hdr.DataOffsetAndFlags&CTRL_ACK != 0, hdr.DataOffsetAndFlags&CTRL_PSH != 0, hdr.DataOffsetAndFlags&CTRL_FIN != 0,
		hdr.Window, hdr.Window, hdr.Checksum, hdr.Checksum, hdr.Checksum, 4*((hdr.DataOffsetAndFlags>>4)&0x0f), 4*((hdr.DataOffsetAndFlags>>4)&0x0f)-20)
}

func Parse(skbuf *skb.SkBuff) {
	skbuf.TransportHdr = util.Byte2Pointer(skbuf.Buffer[skbuf.TransportIndex:])
	hdr := (*TCPHeader)(skbuf.TransportHdr)
	skbuf.PayloadLen -= 4 * ((hdr.DataOffsetAndFlags >> 4) & 0x0f)
	skbuf.AppIndex = skbuf.TransportIndex + (4 * ((hdr.DataOffsetAndFlags >> 4) & 0x0f))
}

func TCP_SKB_CB(__skb *skb.SkBuff) *tcp_skb_cb {
	return (*tcp_skb_cb)(util.Byte2Pointer(__skb.CB[0:40]))
}
