package arp

import (
	"log"
	"testing"

	"github.com/hedwig100/go-network/pkg/device"
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

func Test2ARP(t *testing.T) {
	org_hdr := ArpEther{
		ArpHeader: ArpHeader{
			Hrd: arpHrdEther,
			Pro: arpProIP,
			Hln: device.EtherAddrLen,
			Pln: ip.AddrLen,
			Op:  arpOpReply,
		},
		Sha: device.EtherAddrAny,
		Spa: ip.AddrAny,
		Tha: device.EtherAddrBroadcast,
		Tpa: ip.AddrBroadcast,
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
