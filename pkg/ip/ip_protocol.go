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

var protos []Proto

// Proto is the upper protocol of IP such as TCP,UDP
type Proto interface {

	// Protocol Type
	Type() ProtoType

	// Receive Handler
	RxHandler(data []byte, src Addr, dst Addr, iface *Iface) error
}

// ProtoRegister is used to register ip.Proto
func ProtoRegister(proto Proto) error {

	// check if the same type IpUpperProtocol is already registered
	for _, registerd := range protos {
		if registerd.Type() == proto.Type() {
			return fmt.Errorf("IP protocol(type=%s) is already registerd", proto.Type())
		}
	}

	protos = append(protos, proto)
	log.Printf("[I] registered proto=%s", proto.Type())
	return nil
}
