package tcp

import (
	"log"
	"testing"

	"github.com/hedwig100/go-network/pkg/ip"
)

func compareByte(a []byte, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func Test2TCP(t *testing.T) {
	org_hdr := TCPHeader{
		Src:    80,
		Dst:    20,
		Seq:    9,
		Ack:    101,
		Offset: 124,
		Flag:   ACK,
		Window: 1010,
		Urgent: 0xf1,
	}
	org_payload := []byte{0x99, 0x1e, 0x0a, 0x9c, 0x9f}
	src_, _ := ip.Str2IPAddr("8.8.8.8")
	dst_, _ := ip.Str2IPAddr("192.0.2.2")
	src := ip.IPAddr(src_)
	dst := ip.IPAddr(dst_)

	data, err := header2dataTCP(&org_hdr, org_payload, src, dst)
	if err != nil {
		t.Error(err)
	}

	new_hdr, new_payload, err := data2headerTCP(data, src, dst)
	if err != nil {
		t.Error(err)
	}

	log.Printf("%s\n", org_hdr)
	log.Println(org_payload)
	log.Println(data)
	log.Printf("%s\n", new_hdr)
	log.Println(new_payload)

	if org_hdr != new_hdr {
		t.Error("TCP header transform not succeeded")
	}
	if !compareByte(org_payload, new_payload) {
		t.Error("TCP payload transforrm not succeeded")
	}
}
