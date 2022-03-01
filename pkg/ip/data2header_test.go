package ip

import (
	"log"
	"testing"

	"github.com/hedwig100/go-network/pkg/device"
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
