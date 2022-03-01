package net

import (
	"log"
	"testing"

	"github.com/hedwig100/go-network/net/device"
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

func Test2IP(t *testing.T) {
	src, _ := Str2IPAddr("127.0.0.1")
	dst, _ := Str2IPAddr("8.8.8.8")

	org_hdr := IPHeader{
		Vhl:          IPVersionIPv4<<4 | IPHeaderSizeMin>>2,
		Tos:          0xff,
		Tol:          IPHeaderSizeMin + 3,
		Id:           1,
		Flags:        0,
		Ttl:          64,
		ProtocolType: IPProtocolICMP,
		Checksum:     0,
		Src:          IPAddr(src),
		Dst:          IPAddr(dst),
	}
	org_payload := []byte{0x92, 0x12, 0x29}

	data, err := header2dataIP(&org_hdr, org_payload)
	if err != nil {
		t.Error(err)
	}

	new_hdr, new_payload, err := data2headerIP(data)
	if err != nil {
		t.Error(err)
	}

	log.Printf("%s\n", org_hdr)
	log.Println(org_payload)
	log.Println(data)
	log.Printf("%s\n", new_hdr)
	log.Println(new_payload)

	if org_hdr != new_hdr {
		t.Error("IPv4 header transform not succeeded")
	}
	if !compareByte(org_payload, new_payload) {
		t.Error("IPv4 payload transforrm not succeeded")
	}
}

func Test2ARP(t *testing.T) {
	org_hdr := ArpEther{
		ArpHeader: ArpHeader{
			Hrd: arpHrdEther,
			Pro: arpProIP,
			Hln: device.EtherAddrLen,
			Pln: IPAddrLen,
			Op:  arpOpReply,
		},
		Sha: device.EtherAddrAny,
		Spa: IPAddrAny,
		Tha: device.EtherAddrBroadcast,
		Tpa: IPAddrBroadcast,
	}

	data, err := header2dataARP(org_hdr)
	log.Printf("%v\n", data)
	if err != nil {
		t.Error(err)
	}

	new_hdr, err := data2headerARP(data)
	if err != nil {
		t.Error(err)
	}

	log.Printf("%s\n", org_hdr)
	log.Printf("%v\n", data)
	log.Printf("%s\n", new_hdr)

	if org_hdr != new_hdr {
		t.Error("ARP header transform not succeeded")
	}
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
	src_, _ := Str2IPAddr("8.8.8.8")
	dst_, _ := Str2IPAddr("192.0.2.2")
	src := IPAddr(src_)
	dst := IPAddr(dst_)

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
	src_, _ := Str2IPAddr("8.8.8.8")
	dst_, _ := Str2IPAddr("192.0.2.2")
	src := IPAddr(src_)
	dst := IPAddr(dst_)

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
