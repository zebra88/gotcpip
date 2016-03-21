package socket

import (
	"tcpip/net/skb"
)

const PICO_LOOP_DIR_IN = 1
const PICO_LOOP_DIR_OUT = 2

type uint8 ZBLayer

const (
	ZB_LAYER_DATALINK  = 2 /* Ethernet only. */
	ZB_LAYER_NETWORK   = 3 /* IPv4, IPv6, ARP. Arp is there because it communicates with L2 */
	ZB_LAYER_TRANSPORT = 4 /* UDP, TCP, ICMP */
	ZB_LAYER_SOCKET    = 5 /* Socket management */
)

type ZBProtocol struct {
	name        string
	hash        uint32
	Layer       ZBLayer
	ProtoNumber uint16
	QueueIn     chan *skb.SkBuff
	QueueOut    chan *skb.SkBuff
	//	map[string]interface{}
	//    struct pico_queue *q_in
	//    struct pico_queue *q_out;
	//    struct pico_frame *(*alloc)(struct pico_protocol *self, uint16_t size); /* Frame allocation. */
	//    int (*push)(struct pico_protocol *self, struct pico_frame *p);    /* Push function, for active outgoing pkts from above */
	//    int (*process_out)(struct pico_protocol *self, struct pico_frame *p);  /* Send loop. */
	//    int (*process_in)(struct pico_protocol *self, struct pico_frame *p);  /* Recv loop. */
	//    uint16_t (*get_mtu)(struct pico_protocol *self);
}
