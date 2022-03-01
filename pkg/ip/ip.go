package ip

import (
	"fmt"
	"log"
	"math"

	"github.com/hedwig100/go-network/pkg/device"
	"github.com/hedwig100/go-network/pkg/net"
)

const (
	IPVersionIPv4 = 4
	IPVersionIPv6 = 6

	IPHeaderSizeMin        = 20
	IPPayloadSizeMax       = math.MaxUint16 - IPHeaderSizeMin
	IPAddrLen        uint8 = 4
)

// Init prepares the IP protocol
func Init(done chan struct{}) error {
	arpInit(done)
	err := net.ProtocolRegister(&IPProtocol{})
	return err
}

/*
	IP address
*/

const (
	IPAddrAny       IPAddr = 0x00000000
	IPAddrBroadcast IPAddr = 0xffffffff
)

// IPAddr is IP address
type IPAddr uint32

func (a IPAddr) String() string {
	b := uint32(a)
	return fmt.Sprintf("%d.%d.%d.%d", (b>>24)&0xff, (b>>16)&0xff, (b>>8)&0xff, b&0xff)
}

var id uint16 = 0

// generateId() generates id for IP header
func generateId() uint16 {
	id++
	return id
}

/*
	IP Protocol
*/

// IPProtocol is struct for IP Protocol. This implements protocol interface.
type IPProtocol struct{}

func (p *IPProtocol) Type() net.ProtocolType {
	return net.ProtocolTypeIP
}

// TxHandlerIP receives data from IPUpperProtocol and transmit the data with the device
func TxHandlerIP(protocol ProtoType, data []byte, src IPAddr, dst IPAddr) error {

	// if dst is broadcast address, source is required
	if src == IPAddrAny && dst == IPAddrBroadcast {
		return fmt.Errorf("source is required for broadcast address")
	}

	// look up routing table
	route, err := LookupTable(dst)
	if err != nil {
		return err
	}

	// source address must be the same as interface's one
	ipIface := route.IpIface
	if src != IPAddrAny && src != ipIface.Unicast {
		return fmt.Errorf("unable to output with specified source address,addr=%s", src)
	}

	var nexthop IPAddr
	if route.nexthop != IPAddrAny {
		nexthop = route.nexthop
	} else {
		nexthop = dst
	}

	// does not support fragmentation
	if int(ipIface.dev.MTU()) < IPHeaderSizeMin+len(data) {
		return fmt.Errorf("dst(%v) IP address cannot be reachable(broadcast=%v)", dst, ipIface.broadcast)
	}

	// transform IP header to byte strings
	hdr := Header{
		Vhl:       (IPVersionIPv4<<4 | IPHeaderSizeMin>>2),
		Tol:       uint16(IPHeaderSizeMin + len(data)),
		Id:        generateId(),
		Flags:     0,
		Ttl:       0xff,
		ProtoType: protocol,
		Checksum:  0,
		Src:       ipIface.Unicast,
		Dst:       dst,
	}
	data, err = header2dataIP(&hdr, data)
	if err != nil {
		return err
	}

	// transmit data from the device
	var hwaddr net.HardwareAddr
	if ipIface.dev.Flags()&net.NetDeviceFlagNeedARP > 0 { // check if arp is necessary
		if nexthop == ipIface.broadcast || nexthop == IPAddrBroadcast {
			hwaddr = device.EtherAddrBroadcast // TODO: not only ethernet
		} else {
			hwaddr, err = ArpResolve(ipIface, nexthop)
			if err != nil {
				return err
			}
		}
	}

	log.Printf("[D] IP TxHandler: iface=%d,dev=%s,header=%s", ipIface.Family(), ipIface.dev.Name(), hdr)
	return ipIface.dev.Transmit(data, net.ProtocolTypeIP, hwaddr)
}

func (p *IPProtocol) RxHandler(ch chan net.ProtocolBuffer, done chan struct{}) {
	var pb net.ProtocolBuffer

	for {

		// check if finished or not
		select {
		case <-done:
			return
		default:
		}

		// receive data from device
		pb = <-ch

		// extract the header from the beginning of the data
		ipHdr, payload, err := data2headerIP(pb.Data)
		if err != nil {
			log.Printf("[E] IP rxHandler: %s", err.Error())
			continue
		}

		if ipHdr.Flags&0x2000 > 0 || ipHdr.Flags&0x1fff > 0 {
			log.Printf("[E] IP rxHandler: does not support fragments")
			continue
		}

		// search the interface whose address matches the header's one
		var ipIface *Iface
		var ok bool
		for _, iface := range pb.Dev.Interfaces() {
			ipIface, ok = iface.(*Iface)
			if ok && (ipIface.Unicast == ipHdr.Dst || ipIface.broadcast == IPAddrBroadcast || ipIface.broadcast == ipHdr.Dst) {
				break
			}
		}
		if ipIface == nil {
			log.Printf("[D] IP rxHandler: packet is to other host")
			continue // the packet is to other host
		}
		log.Printf("[D] IP rxHandler: iface=%s,protocol=%s,header=%v", ipIface.Unicast, ipHdr.ProtoType, ipHdr)

		// search the protocol whose type is the same as the header's one
		for _, proto := range Protos {
			if proto.Type() == ipHdr.ProtoType {
				err = proto.RxHandler(payload, ipHdr.Src, ipHdr.Dst, ipIface)
				if err != nil {
					log.Printf("[E] IP RxHanlder: %s", err.Error())
				}
			}
		}
	}
}
