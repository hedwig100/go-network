package net

import "log"

/*
	Protocol Type
*/

const (
	ProtoTypeIP   ProtoType = 0x0800
	ProtoTypeArp  ProtoType = 0x0806
	ProtoTypeIPv6 ProtoType = 0x86dd
)

type ProtoType uint16

func (pt ProtoType) String() string {
	switch pt {
	case ProtoTypeIP:
		return "IPv4"
	case ProtoTypeArp:
		return "ARP"
	case ProtoTypeIPv6:
		return "IPv6"
	default:
		return "UNKNOWN"
	}
}

/*
	Protocol
*/

const ProtoBufferSize = 100

var Protos []Proto
var ProtoBuffers []chan ProtoBuffer

// Proto is the ans abstraction of protocol
type Proto interface {

	// protocol type ex) IP,IPv6,ARP
	Type() ProtoType

	// receive handler
	RxHandler(chan ProtoBuffer, chan struct{})
}

// ProtoBuffer is each protocol's buffer, read the data from here which the device puts
type ProtoBuffer struct {

	// Data from the device
	Data []byte

	// device
	Dev Device
}

// ProtoRegister registers the  protocol
func ProtoRegister(proto Proto) (err error) {

	// add thee protocol
	ch := make(chan ProtoBuffer, ProtoBufferSize)
	Protos = append(Protos, proto)
	ProtoBuffers = append(ProtoBuffers, ch)

	log.Printf("[I] registerd proto=%s", proto.Type())
	return
}
