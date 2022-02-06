package ip

import "fmt"

type IPProtocolType uint8

const (
	IPProtocolICMP IPProtocolType = 1
	IPProtocolTCP  IPProtocolType = 6
	IPProtocolUDP  IPProtocolType = 11
)

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

var IPProtocols []IPUpperProtocol

// Ip Upper Protocol is upper protocol of IP such as TCP,UDP
type IPUpperProtocol interface {

	// Protocol Type
	Type() IPProtocolType

	// Transmit handler
	// TxHandler()

	// Receive Handler
	RxHandler(data []byte, src IPAddr, dst IPAddr, ipIface *IPIface)
}

// IpProtocolRegister is used to register IpUpperProtocol
func IpProtocolRegister(iproto IPUpperProtocol) error {

	// check if the same type IpUpperProtocol is already registered
	for _, proto := range IPProtocols {
		if proto.Type() == iproto.Type() {
			return fmt.Errorf("IP protocol(type=%s) is already registerd", proto.Type())
		}
	}

	IPProtocols = append(IPProtocols, iproto)
	return nil
}
