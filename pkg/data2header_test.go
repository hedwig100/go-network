package pkg

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
func Test2ICMP(t *testing.T) {
	org_hdr := ICMPHeader{
		Typ:      ICMPTypeDestUnreach,
		Code:     ICMPCodeNetUnreach,
		Checksum: 0,
		Values:   19,
	}
	org_payload := []byte{90, 21, 143, 134}

	data, err := header2dataICMP(&org_hdr, org_payload)
	if err != nil {
		t.Error(err)
	}

	new_hdr, new_payload, err := data2headerICMP(data)
	if err != nil {
		t.Error(err)
	}

	log.Printf("%s\n", org_hdr)
	log.Println(org_payload)
	log.Println(data)
	log.Printf("%s\n", new_hdr)
	log.Println(new_payload)

	if org_hdr != new_hdr {
		t.Error("ICMP header transform not succeeded")
	}
	if !compareByte(org_payload, new_payload) {
		t.Error("ICMP payload transforrm not succeeded")
	}
}

func Test2UDP(t *testing.T) {
	org_hdr := UDPHeader{
		Src: 80,
		Dst: 20,
		Len: uint16(UDPHeaderSize + 5),
	}
	org_payload := []byte{0x99, 0x1e, 0x0a, 0x9c, 0x9f}
	src_, _ := ip.Str2IPAddr("8.8.8.8")
	dst_, _ := ip.Str2IPAddr("192.0.2.2")
	src := ip.IPAddr(src_)
	dst := ip.IPAddr(dst_)

	data, err := header2dataUDP(&org_hdr, org_payload, src, dst)
	if err != nil {
		t.Error(err)
	}

	new_hdr, new_payload, err := data2headerUDP(data, src, dst)
	if err != nil {
		t.Error(err)
	}

	log.Printf("%s\n", org_hdr)
	log.Println(org_payload)
	log.Println(data)
	log.Printf("%s\n", new_hdr)
	log.Println(new_payload)

	if org_hdr != new_hdr {
		t.Error("UDP header transform not succeeded")
	}
	if !compareByte(org_payload, new_payload) {
		t.Error("UDP payload transforrm not succeeded")
	}
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
