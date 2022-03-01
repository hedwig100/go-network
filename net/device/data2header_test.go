package device

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
