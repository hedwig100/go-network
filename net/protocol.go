package net

import "log"

type ProtocolType uint16

const (
	ProtocolTypeIP   ProtocolType = 0x0800
	ProtocolTypeArp  ProtocolType = 0x0806
	ProtocolTypeIPv6 ProtocolType = 0x86dd
)

func (pt ProtocolType) String() string {
	switch pt {
	case ProtocolTypeIP:
		return "IP"
	case ProtocolTypeArp:
		return "ARP"
	case ProtocolTypeIPv6:
		return "IPv6"
	default:
		return "UNKNOWN"
	}

}

type Protocol interface {
	Name() string
	Type() ProtocolType

	TxHandler([]byte) error
	RxHandler([]byte, Device) error
}

var Protocols []Protocol

func ProtocolRegister(proto Protocol) (err error) {
	Protocols = append(Protocols, proto)
	log.Printf("registerd dev=%s", proto.Name())
	return
}
