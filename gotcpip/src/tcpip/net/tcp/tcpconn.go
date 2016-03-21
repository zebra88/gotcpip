package tcp

import (
	//	"bufio"
	//	"bytes"
	//	"encoding/binary"
	"fmt"
	"github.com/eapache/channels"
	"log"
	"math"
	"math/rand"
	"slab"
	"tcpip/net/ip"
	"tcpip/net/skb"
	"tcpip/util"
	"unsafe"
)

const (
	STATE_CLOSED       = iota
	STATE_SYN_RECEIVED = iota
	STATE_SYN_SENT     = iota
	STATE_ESTABLISHED  = iota
)

type TCPConn struct {
	localIP    uint32
	localPort  uint16
	remoteIP   uint32
	remotePort uint16
	state      int

	recvWindow  uint16
	recvNextSeq uint32 "RCV.NXT"
	sendNextSeq uint32 "SND.NXT"
	sendWindow  uint16
	unacked     uint32 "SND.UNA"
	sock        TCPSocket
	ipConn      *ip.IPConn
	readChan    channels.Channel
	sem         chan bool
}

type TCPSocket struct {
	snd_wl1    uint32 /* Sequence for window update       */
	snd_wnd    uint32 /* The window we expect to receive  */
	max_window uint32 /* Maximal window ever seen from peer   */
	mss_cache  uint   /* Cached effective mss, not including SACKS */

	window_clamp uint32 /* Maximal window to advertise      */
	rcv_ssthresh uint32 /* Current window clamp         */

	frto_highmark uint32 /* snd_nxt when RTO occurred */
	advmss        uint32 /* Advertised MSS           */
	frto_counter  uint32 /* Number of new acks after RTO */
	nonagle       uint32 /* Disable Nagle algorithm?             */

	/* RTT measurement */
	srtt     uint32 /* smoothed round trip time << 3    */
	mdev     uint32 /* medium deviation         */
	mdev_max uint32 /* maximal mdev for the last rtt period */
	rttvar   uint32 /* smoothed mdev_max            */
	rtt_seq  uint32 /* sequence number to update rttvar */

	packets_out uint32 /* Packets which are "in flight"    */
	retrans_out uint32 /* Retransmitted packets out        */

	urg_data   uint32 /* Saved octet of OOB data and control flags */
	ecn_flags  uint32 /* ECN status bits.         */
	reordering uint32 /* Packet reordering metric.        */
	snd_up     uint32 /* Urgent pointer       */

	keepalive_probes uint32 /* num of allowed keep alive probes   */
	/*
	 *      Options received (usually on last packet, some only on SYN packets).
	 */
	rx_opt TCPOptionsReceived

	/*
	 *  Slow start and congestion control (see also Nagle, and Karn & Partridge)
	 */
	snd_ssthresh   uint32 /* Slow start size threshold        */
	snd_cwnd       uint32 /* Sending congestion window        */
	snd_cwnd_cnt   uint32 /* Linear increase counter      */
	snd_cwnd_clamp uint32 /* Do not allow snd_cwnd to grow above this */
	snd_cwnd_used  uint32
	snd_cwnd_stamp uint32

	rcv_wnd      uint32 /* Current receiver window      */
	write_seq    uint32 /* Tail(+1) of data held in tcp send buffer */
	pushed_seq   uint32 /* Last pushed seq, required to talk to windows */
	lost_out     uint32 /* Lost packets         */
	sacked_out   uint32 /* SACK'd packets           */
	fackets_out  uint32 /* FACK'd packets           */
	tso_deferred uint32
	bytes_acked  uint32 /* Appropriate Byte Counting - RFC3465 */
}

func tcpSackReset(rx_opt *TCPOptionsReceived) {
	rx_opt.dsack = false
	rx_opt.num_sacks = 0
}

var pool *slab.SyncPool

func (conn *TCPConn) ident() string {
	return fmt.Sprintf("%d,%d,%d,%d", conn.localIP, conn.localPort, conn.remoteIP, conn.remotePort)
}

func (conn *TCPConn) Send(skbuf *skb.SkBuff) (written int, err error) {

	// TODO: Pull this from Free List? SYN/ACK's will be set lengths likely
	// So we don't want to keep allocating space when we can just have
	// it pre-allocated
	//tcpPacket.Display()

	// TODO: Store/use []byte version or better an IPAddr object (re-use the stuff from net?)
	written, err = conn.ipConn.Send(skbuf)
	if err != nil {
		log.Fatal(err)
	}
	return
}
func (conn *TCPConn) Read(buf []byte) (read int, err error) {
	amountToRead := int(math.Min(float64(cap(buf)), float64(channels.Buffer(conn.readChan).Len())))

	conn.sem <- true

	for i := 0; i < amountToRead; i++ {
		buf[i] = (<-conn.readChan.Out()).(byte)
	}

	<-conn.sem

	return amountToRead, nil

}

type TCPListener struct {
	localIP     uint32
	localPort   uint16
	connChannel chan *TCPConn
}

// Wait for an incoming connection to the listening IP and Port
func (tl *TCPListener) Accept() (*TCPConn, error) {
	// Wait for a connection in the ESTABLISHED state for the given IP/port combination
	// This in effect means going through the full 3-way Handshake when a SYN packet
	// comes in for the port

	return <-tl.connChannel, nil
}

var connections map[string]*TCPConn
var listeners map[string]*TCPListener

var ipHandle *ip.IPHandle

func Listen(localIP string, port uint16) (*TCPListener, error) {
	ip := util.Ip2long(localIP)
	lChan := make(chan *TCPConn, 5)
	tl := TCPListener{ip, util.Htons(port), lChan}

	listeners[localIP+string(util.Htons(port))] = &tl
	fmt.Printf("Listen %d:%d\n", ip, util.Htons(port))
	return &tl, nil

}

// Because we're using the /dev/tun interface we need to basically bootstrap
// our network and then we can use it as normal
func Start(device string, filter string) error {

	pool = slab.NewSyncPool(
		1500,         // The smallest chunk size is 64B.
		1500*1024*20, // The largest chunk size is 64KB.
		2,            // Power of 2 growth in chunk size.
	)
	fmt.Printf("Start %s\n", device+filter)
	var err error
	inQueue := ip.Start(device, filter)

	if err != nil {
		return err
	} else {
		listeners = make(map[string]*TCPListener)
		connections = make(map[string]*TCPConn)
		go listenForPackets(inQueue)
		return nil
	}

}
func tcpClearOptions(rx_opt *TCPOptionsReceived) {
	rx_opt.tstamp_ok, rx_opt.sack_ok, rx_opt.wscale_ok, rx_opt.snd_wscale = false, 0, false, 0
}
func tcpParseOpt(skbuf *skb.SkBuff, opt_rx *TCPOptionsReceived) int {

	tcpHdr := (*TCPHeader)(skbuf.TransportHdr)
	opt_rx.saw_tstamp = 0
	length := int((4 * ((tcpHdr.DataOffsetAndFlags >> 4) & 0x0f)) - 20)

	//ptr := (*uint8)(util.Byte2Pointer(skbuf.Buffer[skbuf.TransportIndex+20:]))
	opt := skbuf.Buffer[skbuf.TransportIndex+20:]

	index := int(0)
	var opsize int
	for length > 0 {
		opcode := opt[index]
		//		var opsize int
		index++
		switch opcode {
		case TCPOPT_EOL:
			return 0
		case TCPOPT_NOP: /* Ref: RFC 793 section 3.1 */
			length--
			continue
		default:
			opsize = int(opt[index])
			index++
			if opsize < 2 { /* "silly options" */
				return 0
			}
			if opsize > length {
				return 0 /* don't parse partial options */
			}
			log.Print("opcode:", opcode)
			switch opcode {
			case TCPOPT_MSS:
				if opsize == TCPOLEN_MSS && tcpHdr.IsSYN() {
					in_mss := util.Ntohs(util.Byte2uint16(opt[index : index+2])) //  get_unaligned_be16(ptr);
					log.Print("in_mss:", in_mss, opt[index:index+2])
					if in_mss > 0 {
						if opt_rx.user_mss > 0 && opt_rx.user_mss < in_mss {
							in_mss = opt_rx.user_mss
						}
						opt_rx.mss_clamp = in_mss
					}
				}
				break
			case TCPOPT_WINDOW:
				if opsize == TCPOLEN_WINDOW && tcpHdr.IsSYN() {
					snd_wscale := uint8(opt[index])
					opt_rx.wscale_ok = true
					if snd_wscale > 14 {
						log.Print("tcp_parse_options: Illegal window "+
							"scaling value %d >14 received.\n",
							snd_wscale)
						snd_wscale = 14
					}
					opt_rx.snd_wscale = snd_wscale
				}
				break
			case TCPOPT_TIMESTAMP:
				if opsize == TCPOLEN_TIMESTAMP &&
					opt_rx.tstamp_ok == true {
					opt_rx.saw_tstamp = 1
					opt_rx.rcv_tsval = util.Ntohl(util.Byte2uint32(opt[index : index+4]))
					opt_rx.rcv_tsecr = util.Ntohl(util.Byte2uint32(opt[index+4 : index+8]))
				}
				break
			case TCPOPT_SACK_PERM:
				if opsize == TCPOLEN_SACK_PERM && tcpHdr.IsSYN() {
					opt_rx.sack_ok = 1
					tcpSackReset(opt_rx)
				}
				break
			case TCPOPT_SACK:
				if opsize >= (TCPOLEN_SACK_BASE+TCPOLEN_SACK_PERBLOCK) &&
					((opsize-TCPOLEN_SACK_BASE)%TCPOLEN_SACK_PERBLOCK) == 0 &&
					opt_rx.sack_ok > 0 {
					TCP_SKB_CB(skbuf).sacked = 20 + uint8(index-2) //(ptr - 2) - (unsigned char *)th;
				}
				break

			}
			index += opsize - 2
			length -= opsize
		}

	}
	return 0
}

//func tcp_transmit_skb(__skb skb.)
// This is essentially our "run loop"
func listenForPackets(in chan *skb.SkBuff) {
	var err error
	for {
		/*		skbuf, err := ipHandle.Receive()
				if err != nil {
					log.Fatal(err)
				}
		*/
		skbuf := <-in

		ipHdr := (*ip.IPHeader)(skbuf.NetworkHdr)
		Parse(skbuf)

		tcpHdr := (*TCPHeader)(skbuf.TransportHdr)
		tl := listeners[ipHdr.Destination()+string(tcpHdr.DestPort)]
		ident := fmt.Sprintf("%s,%d,%s,%d", ipHdr.Destination(), tcpHdr.DestPort, ipHdr.Source(), tcpHdr.SourcePort)
		fmt.Println(tcpHdr.DestPort)
		fmt.Println(ident)
		tcpConn := connections[ident]

		if tcpConn == nil { // Start of a new connection
			if tcpHdr.IsSYN() && tl != nil {
				fmt.Printf("We have a listener, attempting to establish connection\n")
				tcpConn = &TCPConn{localIP: tl.localIP,
					localPort:  tl.localPort,
					remoteIP:   ipHdr.Saddr,
					remotePort: tcpHdr.SourcePort,
					state:      STATE_SYN_RECEIVED}
				tmp_opt := TCPOptionsReceived{}
				tcpClearOptions(&tmp_opt)
				tmp_opt.mss_clamp = 536
				tcpParseOpt(skbuf, &tmp_opt)

				//tmp_opt.tstamp_ok = tmp_opt.saw_tstamp

				//tcp_v4_save_options

				fmt.Printf("TCP=>Generating an IP Connection\n")
				tcpConn.ipConn, err = ip.NewIPConn(tl.localIP, tcpConn.remoteIP, skbuf)

				if err != nil {
					log.Fatal(err)
				}
				connections[tcpConn.ident()] = tcpConn

			} else if tcpHdr.IsSYN() {
				fmt.Printf("**This is a SYN packet, but we don't have any listeneres!\n")
				continue
				// Send RST
			}
		}

		switch tcpConn.state {
		case STATE_SYN_SENT, STATE_SYN_RECEIVED:
			tcpConn.recvNextSeq = util.Htonl(util.Ntohl(tcpHdr.Seq) + 1)
			tcpConn.recvWindow = tcpHdr.Window
			// This is aprt of congestion management
			if tcpHdr.IsACK() {
				fmt.Println("TCP=>Moving to ESTABLISHED State")
				tcpConn.state = STATE_ESTABLISHED
				tcpConn.readChan = channels.NewNativeChannel(1024 * 10) // 10K
				tcpConn.sem = make(chan bool, 1)
				tl.connChannel <- tcpConn
			} else {
				err = sendSynAck(tcpConn)
				if err != nil {
					fmt.Println("TCP=>Unable to perform SynAck")
					log.Fatal(err)
				}
			}
		case STATE_ESTABLISHED:
			// TODO: We actually have to see how much we were able to write into our buffer
			tcpConn.recvNextSeq = util.Htonl(util.Ntohl(tcpHdr.Seq) + uint32(skbuf.PayloadLen))
			//tcpConn.recvNextSeq = tcpHdr.Seq + uint32(len(tcpPacket.Data))
			tcpConn.recvWindow = tcpHdr.Window // TODO: Figure out our own

			err = sendAck(tcpConn)
			if err != nil {
				log.Fatal(err)
			}

			tcpConn.sem <- true
			fmt.Println("Writing data to read channel")
			for _, val := range skbuf.Buffer[skbuf.AppIndex:] {
				tcpConn.readChan.In() <- val
			}

			<-tcpConn.sem
			fmt.Println("Done writing")

		default:
			fmt.Printf("What're we doing with state %s:%v\n", tcpConn.ident(), tcpConn.state)
		}

	}
}
func initialSequenceNumber() uint32 {
	return rand.Uint32()
}

func sendAck(conn *TCPConn) (err error) {

	skbuf := new(skb.SkBuff)

	skbuf.Buffer = pool.Alloc(54)
	skbuf.TransportIndex = 34
	skbuf.TransportHdr = util.Byte2Pointer(skbuf.Buffer[skbuf.TransportIndex:])

	skbuf.PayloadLen = 20

	tcpHdr := (*TCPHeader)(skbuf.TransportHdr)

	tcpHdr.SourcePort = conn.localPort
	tcpHdr.DestPort = conn.remotePort
	tcpHdr.Seq = conn.sendNextSeq
	dataOffset := uint16(20)
	tcpHdr.ACK(conn.recvNextSeq)

	tcpHdr.DataOffsetAndFlags |= ((dataOffset / 4) << 4) // Number of words
	tcpHdr.Window = conn.recvWindow                      // TODO: Set appropriately
	tcpHdr.Checksum = 0                                  // Should get calcualted automatically?
	tcpHdr.UrgentPointer = 0
	skbuf.DataEnd = skbuf.TransportIndex + 20
	//	ackPacket.OptionsAndPadding = make([]byte, 0) // no options
	//	ackPacket.Data = make([]byte, 0)

	// sequence management
	conn.sendNextSeq = util.Htonl(util.Ntohl(tcpHdr.Seq) + 1)
	//conn.sendNextSeq = tcpHdr.Seq + 1
	conn.unacked = tcpHdr.Seq
	skbuf.NetworkIndex = 14
	//skbuf.Option.OptionLen = 0

	skbuf.CheckPotioner = unsafe.Pointer(&(tcpHdr.Checksum))
	written, err := conn.Send(skbuf)

	fmt.Printf("sendAck Wrote %d bytes with error %v, tcpHdr:%s\n", written, err, tcpHdr)

	return err
}

func sendSynAck(conn *TCPConn) (err error) {

	skbuf := new(skb.SkBuff)

	skbuf.Buffer = pool.Alloc(54)
	skbuf.TransportIndex = 34
	skbuf.TransportHdr = util.Byte2Pointer(skbuf.Buffer[skbuf.TransportIndex:])
	skbuf.PayloadLen = 20
	tcpHdr := (*TCPHeader)(skbuf.TransportHdr)

	tcpHdr.SourcePort = conn.localPort
	tcpHdr.DestPort = conn.remotePort
	tcpHdr.Seq = initialSequenceNumber()
	dataOffset := uint16(20)
	tcpHdr.ACK(conn.recvNextSeq)

	tcpHdr.DataOffsetAndFlags |= CTRL_SYN                // we'll add the data offset
	tcpHdr.DataOffsetAndFlags |= ((dataOffset / 4) << 4) // Number of words
	tcpHdr.Window = conn.recvWindow                      // TODO: Set appropriately
	tcpHdr.Checksum = 0                                  // Should get calcualted automatically?
	tcpHdr.UrgentPointer = 0
	skbuf.DataEnd = skbuf.TransportIndex + 20
	//	ackPacket.OptionsAndPadding = make([]byte, 0) // no options
	//	ackPacket.Data = make([]byte, 0)

	// sequence management
	conn.sendNextSeq = util.Htonl(util.Ntohl(tcpHdr.Seq) + 1)
	//conn.sendNextSeq = tcpHdr.Seq + 1

	conn.unacked = tcpHdr.Seq

	skbuf.NetworkIndex = 14
	//skbuf.Option.OptionLen = 0
	skbuf.CheckPotioner = unsafe.Pointer(&(tcpHdr.Checksum))
	written, err := conn.Send(skbuf)
	fmt.Printf("sendSynAck Wrote %d bytes with error %v\n, -----send tcp----\ntcpHdr:%s\n", written, err, tcpHdr)

	return err
}
