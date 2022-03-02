package arp

import (
	"log"
	"testing"

	"github.com/hedwig100/go-network/pkg/device"
	"github.com/hedwig100/go-network/pkg/ip"
)

func Test2ARP(t *testing.T) {
	org_hdr := ArpEther{
		Header: Header{
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

	data, err := header2data(org_hdr)
	log.Printf("%v\n", data)
	if err != nil {
		t.Error(err)
	}

	new_hdr, err := data2header(data)
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
