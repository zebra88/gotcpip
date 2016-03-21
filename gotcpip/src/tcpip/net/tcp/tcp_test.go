package tcp

import (
	"flag"
	"fmt"
	//	"tcpip/net/ip"
	"testing"
)

func TestRecive(t *testing.T) {
	var device = flag.String("i", "eth0", "device interface")
	var filter = flag.String("f", "tcp and port 65001", "filter specify packet")
	//var ListenIP = flag.String("l", "192.168.1.115", "Listen IP")
	flag.Parse()
	Start(*device, *filter)
	tl, err := Listen("10.211.55.74", 65001)
	if err != nil {
		t.Errorf("Receive packet fail")
	}
	tc, err := tl.Accept()

	if err != nil {
		t.Errorf("Accept packet fail")
	}

	fmt.Println("Connected!")

	// using a "standard" interface
	buf := make([]byte, 1024)

	read := 0
	for {
		read, err = tc.Read(buf)

		if err != nil {
			t.Errorf("Read packet fail")
		}

		if read > 0 {
			fmt.Printf("Read: %s\n", string(buf[:read]))
		}
	}

}
