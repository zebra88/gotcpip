package ethernet

import (
	"flag"
	"time"
	//"tcpip/net/ethernet"
	"testing"
)

func TestRecive(t *testing.T) {
	var device = flag.String("i", "eth0", "device interface")
	var filter = flag.String("f", "tcp and port 65001", "filter specify packet")
	flag.Parse()
	/*packethandle, err := Start(*device, *filter)
	if err != nil {
		t.Errorf("Start Interfac fail")
	_, err = packethandle.Receive()
	if err != nil {
		t.Errorf("Receive packet fail")
	}

	}
	*/
	Start(*device, *filter)
	//packethandle.Send(data)
	for {
		time.After(10 * time.Minute)
	}
}
