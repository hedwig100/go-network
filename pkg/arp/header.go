package arp

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/hedwig100/go-network/pkg/device"
	"github.com/hedwig100/go-network/pkg/ip"
)

// Header is the header for arp protocol
type Header struct {

	// hardware type
	Hrd uint16

	// protocol type
	Pro uint16

	// hardware address length
	Hln uint8

	// protocol address length
	Pln uint8

	// opcode
	Op uint16
}

// ArpEther is arp header for IPv4 and Ethernet
type ArpEther struct {

	// basic info for arp header
	Header

	// source hardware address
	Sha device.EtherAddr

	// source protocol address
	Spa ip.Addr

	// target hardware address
	Tha device.EtherAddr

	// target protocol address
	Tpa ip.Addr
}

func (ae ArpEther) String() string {

	var Hrd string
	switch ae.Hrd {
	case arpHrdEther:
		Hrd = fmt.Sprintf("Ethernet(%d)", arpHrdEther)
	default:
		Hrd = "UNKNOWN"
	}

	var Pro string
	switch ae.Pro {
	case arpProIP:
		Pro = fmt.Sprintf("IPv4(%d)", arpProIP)
	default:
		Pro = "UNKNOWN"
	}

	var Op string
	switch ae.Op {
	case arpOpReply:
		Op = fmt.Sprintf("Reply(%d)", arpOpReply)
	case arpOpRequest:
		Op = fmt.Sprintf("Request(%d)", arpOpRequest)
	default:
		Op = "UNKNOWN"
	}

	return fmt.Sprintf(`
		Hrd: %s,
		Pro: %s,
		Hln: %d,
		Pln: %d,
		Op: %s,
		Sha: %s,
		Spa: %s,
		Tha: %s,
		Tpa: %s,
	`, Hrd, Pro, ae.Hln, ae.Pln, Op, ae.Sha, ae.Spa, ae.Tha, ae.Tpa)
}

// data2header receives data and returns ARP header,the rest of data,error
// now this function only supports IPv4 and Ethernet address resolution
func data2header(data []byte) (ArpEther, error) {

	// only supports IPv4 and Ethernet address resolution
	if len(data) < int(ArpEtherSize) {
		return ArpEther{}, fmt.Errorf("data size is too small for arp header")
	}

	// read header in bigEndian
	var hdr ArpEther
	r := bytes.NewReader(data)
	err := binary.Read(r, binary.BigEndian, &hdr)
	if err != nil {
		return ArpEther{}, err
	}

	// only receive IPv4 and Ethernet
	if hdr.Hrd != arpHrdEther || hdr.Hln != device.EtherAddrLen {
		return ArpEther{}, fmt.Errorf("arp resolves only Ethernet address")
	}
	if hdr.Pro != arpProIP || hdr.Pln != ip.AddrLen {
		return ArpEther{}, fmt.Errorf("arp only supports IP address")
	}
	return hdr, nil
}

func header2data(hdr ArpEther) ([]byte, error) {

	// write data in bigDndian
	var w bytes.Buffer
	err := binary.Write(&w, binary.BigEndian, hdr)

	return w.Bytes(), err
}
