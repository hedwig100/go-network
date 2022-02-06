package ip

import "fmt"

type IpProtocolType uint8

const (
	IpProtocolICMP IpProtocolType = 1
	IpProtocolTCP  IpProtocolType = 6
	IpProtocolUDP  IpProtocolType = 11
)

func (p IpProtocolType) String() string {
	switch p {
	case IpProtocolICMP:
		return "ICMP"
	case IpProtocolTCP:
		return "TCP"
	case IpProtocolUDP:
		return "UDP"
	default:
		return "UNKNOWN"
	}
}

var IpProtocols []IpUpperProtocol

// Ip Upper Protocol is upper protocol of IP such as TCP,UDP
type IpUpperProtocol interface {

	// Protocol Type
	Type() IpProtocolType

	// Transmit handler
	// TxHandler()

	// Receive Handler
	RxHandler(data []byte, src IpAddr, dst IpAddr, ipIface *IpIface)
}

// IpProtocolRegister is used to register IpUpperProtocol
func IpProtocolRegister(iproto IpUpperProtocol) error {

	// check if the same type IpUpperProtocol is already registered
	for _, proto := range IpProtocols {
		if proto.Type() == iproto.Type() {
			return fmt.Errorf("IP protocol(type=%s) is already registerd", proto.Type())
		}
	}

	IpProtocols = append(IpProtocols, iproto)
	return nil
}
