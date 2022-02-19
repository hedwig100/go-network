package net

import (
	"fmt"
	"log"
)

const (
	IPProtocolICMP IPProtocolType = 0x01
	IPProtocolTCP  IPProtocolType = 0x06
	IPProtocolUDP  IPProtocolType = 0x11
)

/*
	IP Protoocol Type is type of the upper protocol of IP
*/

type IPProtocolType uint8

func (p IPProtocolType) String() string {
	switch p {
	case IPProtocolICMP:
		return "ICMP"
	case IPProtocolTCP:
		return "TCP"
	case IPProtocolUDP:
		return "UDP"
	default:
		return "UNKNOWN"
	}
}

/*
	IP Protocols
*/

var IPProtocols []IPUpperProtocol

// IP Upper Protocol is the upper protocol of IP such as TCP,UDP
type IPUpperProtocol interface {

	// Protocol Type
	Type() IPProtocolType

	// Receive Handler
	rxHandler(data []byte, src IPAddr, dst IPAddr, ipIface *IPIface) error
}

// IPProtocolRegister is used to register IpUpperProtocol
func IPProtocolRegister(iproto IPUpperProtocol) error {

	// check if the same type IpUpperProtocol is already registered
	for _, proto := range IPProtocols {
		if proto.Type() == iproto.Type() {
			return fmt.Errorf("IP protocol(type=%s) is already registerd", proto.Type())
		}
	}

	IPProtocols = append(IPProtocols, iproto)
	log.Printf("[I] registered proto=%s", iproto.Type())
	return nil
}
