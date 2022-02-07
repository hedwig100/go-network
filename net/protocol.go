package net

import "log"

type ProtocolType uint16

const (
	ProtocolTypeIP   ProtocolType = 0x0800
	ProtocolTypeArp  ProtocolType = 0x0806
	ProtocolTypeIPv6 ProtocolType = 0x86dd

	ProtocolBufferSize = 100
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

var Protocols []Protocol
var ProtocolBuffers []chan ProtocolBuffer

// Protocol is the eabstraction of protocol
type Protocol interface {

	// name
	Name() string

	// protocol type ex) IP,IPv6,ARP
	Type() ProtocolType

	// transmit handler
	// TxHandler([]byte) error

	// reeceive handler
	RxHandler(chan ProtocolBuffer, chan struct{})
}

// ProtocolBuffer is each protocol's buffer, read the data from here which the device puts
type ProtocolBuffer struct {
	Data []byte
	Dev  Device
}

// ProtocolRegister registers the  protocol
func ProtocolRegister(proto Protocol) (err error) {

	// add thee protocol
	ch := make(chan ProtocolBuffer, ProtocolBufferSize)
	Protocols = append(Protocols, proto)
	ProtocolBuffers = append(ProtocolBuffers, ch)

	// activate the receive handler
	go proto.RxHandler(ch, done)
	log.Printf("registerd dev=%s", proto.Name())
	return
}
