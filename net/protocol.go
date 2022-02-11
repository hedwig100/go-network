package net

import "log"

/*
	Protocol Type
*/

const (
	ProtocolTypeIP   ProtocolType = 0x0800
	ProtocolTypeArp  ProtocolType = 0x0806
	ProtocolTypeIPv6 ProtocolType = 0x86dd
)

type ProtocolType uint16

func (pt ProtocolType) String() string {
	switch pt {
	case ProtocolTypeIP:
		return "IPv4"
	case ProtocolTypeArp:
		return "ARP"
	case ProtocolTypeIPv6:
		return "IPv6"
	default:
		return "UNKNOWN"
	}
}

/*
	Protocol
*/

const ProtocolBufferSize = 100

var Protocols []Protocol
var ProtocolBuffers []chan ProtocolBuffer

// Protocol is the eabstraction of protocol
type Protocol interface {

	// protocol type ex) IP,IPv6,ARP
	Type() ProtocolType

	// receive handler
	RxHandler(chan ProtocolBuffer, chan struct{})
}

// ProtocolBuffer is each protocol's buffer, read the data from here which the device puts
type ProtocolBuffer struct {

	// Data from the device
	Data []byte

	// device
	Dev Device
}

// ProtocolRegister registers the  protocol
func ProtocolRegister(proto Protocol) (err error) {

	// add thee protocol
	ch := make(chan ProtocolBuffer, ProtocolBufferSize)
	Protocols = append(Protocols, proto)
	ProtocolBuffers = append(ProtocolBuffers, ch)

	log.Printf("[I] registerd proto=%s", proto.Type())
	return
}
