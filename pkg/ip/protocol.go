package ip

import (
	"fmt"
	"log"
	"math"

	"github.com/hedwig100/go-network/pkg/device"
	"github.com/hedwig100/go-network/pkg/net"
)

const (
	V4 = 4
	V6 = 6

	HeaderSizeMin        = 20
	PayloadSizeMax       = math.MaxUint16 - HeaderSizeMin
	AddrLen        uint8 = 4
)

var (
	// NOTE: resolver is arp.ArpResolver
	resolve func(net.Interface, Addr) (net.HardwareAddr, error)
)

// Init prepares the IP protocol
// this receives arp.Resolver
func Init(resolver func(net.Interface, Addr) (net.HardwareAddr, error)) error {
	resolve = resolver
	err := net.ProtoRegister(&IProto{})
	return err
}

/*
	IP address
*/

const (
	AddrAny       Addr = 0x00000000
	AddrBroadcast Addr = 0xffffffff
)

// Addr is IP address
type Addr uint32

func (a Addr) String() string {
	b := uint32(a)
	return fmt.Sprintf("%d.%d.%d.%d", (b>>24)&0xff, (b>>16)&0xff, (b>>8)&0xff, b&0xff)
}

/*
	IP Protocol
*/

// IProto is struct for IP Protocol. This implements protocol interface.
type IProto struct{}

func (p *IProto) Type() net.ProtoType {
	return net.ProtoTypeIP
}

// TxHandler receives data from IPUpperProtocol and transmit the data with the device
func TxHandler(proto ProtoType, data []byte, src Addr, dst Addr) error {

	// if dst is broadcast address, source is required
	if src == AddrAny && dst == AddrBroadcast {
		return fmt.Errorf("source is required for broadcast address")
	}

	// look up routing table
	route, err := LookupTable(dst)
	if err != nil {
		return err
	}

	// source address must be the same as interface's one
	iface := route.Iface
	if src != AddrAny && src != iface.Unicast {
		return fmt.Errorf("unable to output with specified source address,addr=%s", src)
	}

	var nexthop Addr
	if route.nexthop != AddrAny {
		nexthop = route.nexthop
	} else {
		nexthop = dst
	}

	// does not support fragmentation
	if int(iface.dev.MTU()) < HeaderSizeMin+len(data) {
		return fmt.Errorf("dst(%v) IP address cannot be reachable(broadcast=%v)", dst, iface.broadcast)
	}

	// transform IP header to byte strings
	hdr := Header{
		Vhl:       (V4<<4 | HeaderSizeMin>>2),
		Tol:       uint16(HeaderSizeMin + len(data)),
		Id:        generateId(),
		Flags:     0,
		Ttl:       0xff,
		ProtoType: proto,
		Checksum:  0,
		Src:       iface.Unicast,
		Dst:       dst,
	}
	data, err = header2data(&hdr, data)
	if err != nil {
		return err
	}

	// transmit data from the device
	var hwaddr net.HardwareAddr
	if iface.dev.Flags()&net.DeviceFlagNeedARP > 0 { // check if arp is necessary
		if nexthop == iface.broadcast || nexthop == AddrBroadcast {
			hwaddr = device.EtherAddrBroadcast // TODO: not only ethernet
		} else {
			hwaddr, err = resolve(iface, nexthop) // NOTE: resolver is arp.ArpResolver
			if err != nil {
				return err
			}
		}
	}

	log.Printf("[D] IP TxHandler: iface=%d,dev=%s,header=%s", iface.Family(), iface.dev.Name(), hdr)
	return net.DeviceOutput(iface.dev, data, net.ProtoTypeIP, hwaddr)
}

func (p *IProto) RxHandler(ch chan net.ProtoBuffer, done chan struct{}) {
	var pb net.ProtoBuffer

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
		hdr, payload, err := data2header(pb.Data)
		if err != nil {
			log.Printf("[E] IP rxHandler: %s", err.Error())
			continue
		}

		if hdr.Flags&0x2000 > 0 || hdr.Flags&0x1fff > 0 {
			log.Printf("[E] IP rxHandler: does not support fragments")
			continue
		}

		// search the interface whose address matches the header's one
		var iface *Iface
		var ok bool
		for _, candidate := range pb.Dev.Interfaces() {
			iface, ok = candidate.(*Iface)
			if ok && (iface.Unicast == hdr.Dst || iface.broadcast == AddrBroadcast || iface.broadcast == hdr.Dst) {
				break
			}
		}
		if iface == nil {
			log.Printf("[D] IP rxHandler: packet is to other host")
			continue // the packet is to other host
		}
		log.Printf("[D] IP rxHandler: iface=%s,protocol=%s,header=%v", iface.Unicast, hdr.ProtoType, hdr)

		// search the protocol whose type is the same as the header's one
		for _, proto := range protos {
			if proto.Type() == hdr.ProtoType {
				err = proto.RxHandler(payload, hdr.Src, hdr.Dst, iface)
				if err != nil {
					log.Printf("[E] IP RxHanlder: %s", err.Error())
				}
			}
		}
	}
}
