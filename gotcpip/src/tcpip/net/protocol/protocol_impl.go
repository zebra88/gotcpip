package protocol

import (
	"constant"
	"hash/crc32"
	"log"
	"rbtree"
	"tcpip/net/skb"
)

type ZBProtoRR struct {
	rbtree          *rbtree.RBtree
	nodeIn, nodeOut *rbtree.Node
}

var ProtoRRDatalink, ProtoRRNetwork, ProtoRRTransport, ProtoRRSocket ZBProtoRR

func ProtocolInit(p *ZBProtocol) {

	if p == nil {
		return
	}

	DatalinkProtoTree := rbtree.InitTree(protocmp)
	if DatalinkProtoTree == nil {
		log.Println("Init Datalink protocol tree fail")
	}

	NetworkProtoTree := rbtree.InitTree(protocmp)
	if NetworkProtoTree == nil {
		log.Println("Init Network protocol tree fail")
	}

	TransportProtoTree := rbtree.InitTree(protocmp)
	if TransportProtoTree == nil {
		log.Println("Init Transport protocol tree fail")
	}

	SocketProtoTree := rbtree.InitTree(protocmp)
	if TransportProtoTree == nil {
		log.Println("Init Transport protocol tree fail")
	}

	ProtoRRDatalink.rbtree = DatalinkProtoTree
	ProtoRRNetwork.rbtree = NetworkProtoTree
	ProtoRRTransport.rbtree = TransportProtoTree
	ProtoRRTransport.rbtree = SocketProtoTree

	h := crc32.NewIEEE()
	h.Write([]byte(p.Name))
	p.Hash = h.Sum32()
	switch p.Layer {
	case constant.ZB_LAYER_DATALINK:
		DatalinkProtoTree.Insert(p)
		ProtoRRDatalink.nodeIn = nil
		ProtoRRDatalink.nodeOut = nil
		break
	case constant.ZB_LAYER_NETWORK:
		NetworkProtoTree.Insert(p)
		ProtoRRNetwork.nodeIn = nil
		ProtoRRNetwork.nodeOut = nil
		break
	case constant.ZB_LAYER_TRANSPORT:
		TransportProtoTree.Insert(p)
		ProtoRRTransport.nodeIn = nil
		ProtoRRTransport.nodeOut = nil
		break
	case constant.ZB_LAYER_SOCKET:
		SocketProtoTree.Insert(p)
		ProtoRRSocket.nodeIn = nil
		ProtoRRSocket.nodeOut = nil
		break
	}
	log.Printf("Protocol %s registered (layer: %d).\n", p.Name, p.Layer)

}

const (
	DataLinkProtocol = iota
	NetworkProtocol
	TransportProtocol
	SocketProtocol
)

func ProtocolLoop(loop_score, direction int, layer uint8) int {
	switch layer {
	case DataLinkProtocol:
		return ZBProtocolGenericLoop(&ProtoRRDatalink, loop_score, direction)
	case NetworkProtocol:
		return ZBProtocolGenericLoop(&ProtoRRNetwork, loop_score, direction)
	case TransportProtocol:
		return ZBProtocolGenericLoop(&ProtoRRTransport, loop_score, direction)
	case SocketProtocol:
		return ZBProtocolGenericLoop(&ProtoRRSocket, loop_score, direction)
	}
	log.Println("invaild protocol layer")
	return -1
}

func roundrobinInit(rr *ZBProtoRR, direction int) (nextNode *rbtree.Node) {

	if rr.nodeIn == nil {
		rr.nodeIn = rbtree.TreeFirstNode(rr.rbtree.Root)
	}

	if rr.nodeOut == nil {
		rr.nodeOut = rbtree.TreeFirstNode(rr.rbtree.Root)
	}

	if direction == LOOP_DIR_IN {
		nextNode = rr.nodeIn
	} else {
		nextNode = rr.nodeOut
	}

	return nextNode
}

func roundrobinEnd(rr *ZBProtoRR, direction int, last *rbtree.Node) {
	if direction == LOOP_DIR_IN {
		rr.nodeIn = last
	} else {
		rr.nodeOut = last
	}
}

func ZBProtocolGenericLoop(rr *ZBProtoRR, loopScore int, direction int) int {

	nextNode := roundrobinInit(rr, direction)

	if nextNode == nil {
		return loopScore
	}

	next := nextNode.KeyValue

	/* init start node */
	start := next

	/* round-robin all layer protocols, break if traversed all protocols */
	for loopScore > 1 && next != nil {
		loopScore = protoLoop(next.(*ZBProtocol), loopScore, direction)
		nextNode = rbtree.TreeNext(nextNode)
		next = nextNode.KeyValue
		if next == nil {
			nextNode = rbtree.TreeFirstNode(rr.rbtree.Root)
			next = nextNode.KeyValue
		}

		if next == start {
			break
		}
	}
	roundrobinEnd(rr, direction, nextNode)
	return loopScore
}

func protoLoopIn(proto *ZBProtocol, loopScore int) int {

	for loopScore > 0 {
		if proto.QueueIn == nil || skb.SkbQueueEmpty(proto.QueueIn) {
			break
		}

		_skb := skb.SkbDeQueue(proto.QueueIn)
		if _skb != nil && proto.ProcessIn(proto, _skb) > 0 {
			loopScore--
		}
	}
	return loopScore
}

func protoLoopOut(proto *ZBProtocol, loopScore int) int {

	for loopScore > 0 {
		if proto.QueueOut == nil || skb.SkbQueueEmpty(proto.QueueOut) {
			break
		}

		_skb := skb.SkbDeQueue(proto.QueueOut)
		if _skb != nil && proto.ProcessOut(proto, _skb) > 0 {
			loopScore--
		}
	}
	return loopScore
}

func protoLoop(proto *ZBProtocol, loopScore, direction int) int {

	if direction == LOOP_DIR_IN {
		loopScore = protoLoopIn(proto, loopScore)
	} else if direction == LOOP_DIR_OUT {
		loopScore = protoLoopOut(proto, loopScore)
	}

	return loopScore
}

/*
func ProtocolDatalinkLoop(loop_score, direction int) int {
	return ZBProtocolGenericLoop(&ProtoRRDatalink, loop_score, direction)
}

func ProtocolNetworkLoop(loop_score, direction int) int {
	return ZBProtocolGenericLoop(&ProtoRRNetwork, loop_score, direction)
}
func ProtocolTransportLoop(loop_score, direction int) int {
	return ZBProtocolGenericLoop(&ProtoRRTransport, loop_score, direction)
}

func ProtocolSocketLoop(loop_score, direction int) int {
	return ZBProtocolGenericLoop(&ProtoRRSocket, loop_score, direction)
}
*/
func protocmp(ka, kb rbtree.Item) int {

	var a, b *ZBProtocol
	switch v := ka.(type) {
	case *ZBProtocol:
		a = v
	default:

		log.Println("unknown")
	}
	switch v := kb.(type) {
	case *ZBProtocol:
		b = v
	default:
		log.Println("unknown")
	}

	//	fmt.Printf("======a:%p-%d,b:%p-%d\n", a, a, b, b)
	if a.Hash < b.Hash {
		return -1
	}
	if a.Hash > b.Hash {
		return -1
	} else {
		return 0
	}

}
