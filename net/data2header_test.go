package net

import (
	"log"
	"testing"
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
func Test2Ether(t *testing.T) {
	org_hdr := EthernetHdr{
		Src:  EthernetAddress([EtherAddrLen]byte{0xfb, 0x98, 0xfe, 0x92, 0x9e}),
		Dst:  EthernetAddress([EtherAddrLen]byte{0xf2, 0x90, 0x1d, 0x4e, 0x0a}),
		Type: ProtocolTypeIP,
	}
	org_payload := []byte{0x92, 0x12, 0x29}

	data, err := header2dataEther(org_hdr, org_payload)
	if err != nil {
		t.Error(err)
	}

	new_hdr, new_payload, err := data2headerEther(data)
	if err != nil {
		t.Error(err)
	}

	log.Printf("%s\n", org_hdr)
	log.Println(org_payload)
	log.Println(data)
	log.Printf("%s\n", new_hdr)
	log.Println(new_payload)

	if org_hdr != new_hdr {
		t.Error("Ethernet header transform not succeeded")
	}
	if !compareByte(org_payload, new_payload) {
		t.Error("Ethernet payload transforrm not succeeded")
	}
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
			Hrd: ArpHrdEther,
			Pro: ArpProIP,
			Hln: EtherAddrLen,
			Pln: IPAddrLen,
			Op:  ArpOpReply,
		},
		Sha: EtherAddrAny,
		Spa: IPAddrAny,
		Tha: EtherAddrBroadcast,
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
