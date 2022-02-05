package ip

type IpProtocolType uint8

const (
	IpProtocolICMP IpProtocolType = 1
	IpProtocolTCP  IpProtocolType = 6
	IpProtocolUDP  IpProtocolType = 11
)

type IPHihgerProtocol interface {
}
