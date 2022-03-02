package arp

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"time"

	"github.com/hedwig100/go-network/pkg/device"
	"github.com/hedwig100/go-network/pkg/ip"
	"github.com/hedwig100/go-network/pkg/net"
)

const (
	arpHrdEther uint16 = 0x0001
	arpProIP    uint16 = 0x0800

	ArpHeaderSizeMin uint8 = 8
	ArpEtherSize     uint8 = ArpHeaderSizeMin + 2*device.EtherAddrLen + 2*ip.AddrLen

	arpOpRequest uint16 = 1
	arpOpReply   uint16 = 2
)

// Init prepare the ARP protocol.
func Init(done chan struct{}) error {
	go arpTimer(done)
	err := net.ProtoRegister(&arpProtocol{})
	if err != nil {
		return err
	}
	return nil
}

/*
	Arp Header
*/

// ArpHeader is the header for arp protocol
type ArpHeader struct {

	// hardware type
	Hrd uint16

	// protocol type
	Pro uint16

	// hardware address length
	Hln uint8

	// protocol address length
	Pln uint8

	// opcode
	Op uint16
}

// ArpEther is arp header for IPv4 and Ethernet
type ArpEther struct {

	// basic info for arp header
	ArpHeader

	// source hardware address
	Sha device.EtherAddr

	// source protocol address
	Spa ip.Addr

	// target hardware address
	Tha device.EtherAddr

	// target protocol address
	Tpa ip.Addr
}

func (ae ArpEther) String() string {

	var Hrd string
	switch ae.Hrd {
	case arpHrdEther:
		Hrd = fmt.Sprintf("Ethernet(%d)", arpHrdEther)
	default:
		Hrd = "UNKNOWN"
	}

	var Pro string
	switch ae.Pro {
	case arpProIP:
		Pro = fmt.Sprintf("IPv4(%d)", arpProIP)
	default:
		Pro = "UNKNOWN"
	}

	var Op string
	switch ae.Op {
	case arpOpReply:
		Op = fmt.Sprintf("Reply(%d)", arpOpReply)
	case arpOpRequest:
		Op = fmt.Sprintf("Request(%d)", arpOpRequest)
	default:
		Op = "UNKNOWN"
	}

	return fmt.Sprintf(`
		Hrd: %s,
		Pro: %s,
		Hln: %d,
		Pln: %d,
		Op: %s,
		Sha: %s,
		Spa: %s,
		Tha: %s,
		Tpa: %s,
	`, Hrd, Pro, ae.Hln, ae.Pln, Op, ae.Sha, ae.Spa, ae.Tha, ae.Tpa)
}

// data2ArpHeaderARP receives data and returns ARP header,the rest of data,error
// now this function only supports IPv4 and Ethernet address resolution
func data2headerARP(data []byte) (ArpEther, error) {

	// only supports IPv4 and Ethernet address resolution
	if len(data) < int(ArpEtherSize) {
		return ArpEther{}, fmt.Errorf("data size is too small for arp header")
	}

	// read header in bigEndian
	var hdr ArpEther
	r := bytes.NewReader(data)
	err := binary.Read(r, binary.BigEndian, &hdr)
	if err != nil {
		return ArpEther{}, err
	}

	// only receive IPv4 and Ethernet
	if hdr.Hrd != arpHrdEther || hdr.Hln != device.EtherAddrLen {
		return ArpEther{}, fmt.Errorf("arp resolves only Ethernet address")
	}
	if hdr.Pro != arpProIP || hdr.Pln != ip.AddrLen {
		return ArpEther{}, fmt.Errorf("arp only supports IP address")
	}
	return hdr, nil
}

func header2dataARP(hdr ArpEther) ([]byte, error) {

	// write data in bigDndian
	var w bytes.Buffer
	err := binary.Write(&w, binary.BigEndian, hdr)

	return w.Bytes(), err
}

/*
	Arp Protocol
*/

// arpProtocol implements net.Protocol interface.
type arpProtocol struct{}

func (p *arpProtocol) Type() net.ProtoType {
	return net.ProtoTypeArp
}

func (p *arpProtocol) RxHandler(ch chan net.ProtoBuffer, done chan struct{}) {
	var pb net.ProtoBuffer

	for {

		// check if finished or not
		select {
		case <-done:
			return
		default:
		}

		// receive data from device and transform it to header
		pb = <-ch
		hdr, err := data2headerARP(pb.Data)
		if err != nil {
			log.Printf("[E] ARP rxHandler: %s", err.Error())
		}

		// update arp cache table
		mutex.Lock()
		merge := arpCacheUpdate(hdr.Spa, hdr.Sha)
		mutex.Unlock()

		// search the IP interface of the device
		iface, err := net.GetIface(pb.Dev, net.IfaceFamilyIP)
		if err != nil {
			return // the data is to other host
		}
		ipIface := iface.(*ip.Iface)
		if ipIface == nil || ipIface.Unicast != hdr.Tpa {
			return // the data is to other host
		}

		// insert cache entry if entry is not updated before
		if !merge {
			mutex.Lock()
			arpCacheInsert(hdr.Spa, hdr.Sha)
			mutex.Unlock()
		}

		log.Printf("[D] ARP rxHandler: dev=%s,arp header=%s", pb.Dev.Name(), hdr)

		if hdr.Op == arpOpRequest {
			err = ArpReply(ipIface, hdr.Sha, hdr.Spa, hdr.Sha) // reply arp message
			if err != nil {
				log.Printf("[E] ARP rxHandler: %s", err.Error())
			}
		}
	}
}

// ArpReply transmits ARP reply data to dst
func ArpReply(ipIface *ip.Iface, tha device.EtherAddr, tpa ip.Addr, dst device.EtherAddr) error {

	dev, ok := ipIface.Dev().(*device.Ether)
	if !ok {
		return fmt.Errorf("arp only supports EthernetDevice")
	}

	// create arp header
	rep := ArpEther{
		ArpHeader: ArpHeader{
			Hrd: arpHrdEther,
			Pro: arpProIP,
			Hln: device.EtherAddrLen,
			Pln: ip.AddrLen,
			Op:  arpOpReply,
		},
		Sha: dev.EtherAddr,
		Spa: ipIface.Unicast,
		Tha: tha,
		Tpa: tpa,
	}

	data, err := header2dataARP(rep)
	if err != nil {
		return err
	}

	log.Printf("[D] ARP TxHandler(reply): dev=%s,arp header=%s", dev.Name(), rep)
	return net.DeviceOutput(dev, data, net.ProtoTypeArp, dst)
}

// ArpResolve receives protocol address and returns hardware address
func ArpResolve(iface net.Interface, pa ip.Addr) (net.HardwareAddr, error) {

	// only supports IPv4 and Ethernet protocol
	ipIface, ok := iface.(*ip.Iface)
	if !ok {
		return nil, fmt.Errorf("unsupported protocol address type")
	}
	if ipIface.Dev().Type() != net.DeviceTypeEther {
		return nil, fmt.Errorf("unsupported hardware address type")
	}

	// search cache table
	mutex.Lock()
	index, err := arpCacheSelect(pa)

	// cache not found
	if err != nil {

		index = arpCacheAlloc()
		caches[index] = arpCacheEntry{
			state:   arpCacheStateImcomplete,
			pa:      pa,
			timeval: time.Now(),
		}

		// if cache is not in the table, transmit arp request
		ArpRequest(ipIface, pa)
		mutex.Unlock()

		return nil, err
	}

	// cache found but imcomplete request
	if caches[index].state == arpCacheStateImcomplete {

		// if found cache is imcomplete,it might be a packet loss,so transmit arp request
		ArpRequest(ipIface, pa)
		mutex.Unlock()

		return nil, fmt.Errorf("cache state is imcomplete")
	}

	// cache found and get hardware address
	ha := caches[index].ha
	mutex.Unlock()
	return ha, nil
}

// ArpRequest receives interface and target IP address and transmits ARP request to the host(tpa)
func ArpRequest(ipIface *ip.Iface, tpa ip.Addr) error {

	dev, ok := ipIface.Dev().(*device.Ether)
	if !ok {
		return fmt.Errorf("arp only supports EthernetDevice")
	}

	// create arp header
	rep := ArpEther{
		ArpHeader: ArpHeader{
			Hrd: arpHrdEther,
			Pro: arpProIP,
			Hln: device.EtherAddrLen,
			Pln: ip.AddrLen,
			Op:  arpOpRequest,
		},
		Sha: dev.EtherAddr,
		Spa: ipIface.Unicast,
		Tha: device.EtherAddrAny,
		Tpa: tpa,
	}

	data, err := header2dataARP(rep)
	if err != nil {
		return err
	}

	log.Printf("[D] ARP TxHandler(request): dev=%s,arp header=%s", dev.Name(), rep)
	return net.DeviceOutput(dev, data, net.ProtoTypeArp, device.EtherAddrBroadcast)
}
