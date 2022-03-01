package ip

import (
	"fmt"
	"log"
)

const (
	ProtoICMP ProtoType = 0x01
	ProtoTCP  ProtoType = 0x06
	ProtoUDP  ProtoType = 0x11
)

/*
	ProtoType is type of the upper protocol of IP
*/

type ProtoType uint8

func (p ProtoType) String() string {
	switch p {
	case ProtoICMP:
		return "ICMP"
	case ProtoTCP:
		return "TCP"
	case ProtoUDP:
		return "UDP"
	default:
		return "UNKNOWN"
	}
}

/*
	IP Protocols
*/

var Protos []Proto

// Proto is the upper protocol of IP such as TCP,UDP
type Proto interface {

	// Protocol Type
	Type() ProtoType

	// Receive Handler
	RxHandler(data []byte, src IPAddr, dst IPAddr, ipIface *Iface) error
}

// ProtoRegister is used to register ip.Proto
func ProtoRegister(iproto Proto) error {

	// check if the same type IpUpperProtocol is already registered
	for _, proto := range Protos {
		if proto.Type() == iproto.Type() {
			return fmt.Errorf("IP protocol(type=%s) is already registerd", iproto.Type())
		}
	}

	Protos = append(Protos, iproto)
	log.Printf("[I] registered proto=%s", iproto.Type())
	return nil
}
