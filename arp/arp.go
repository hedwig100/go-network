package arp

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"

	"github.com/hedwig100/go-network/devices"
	"github.com/hedwig100/go-network/ip"
	"github.com/hedwig100/go-network/net"
)

const (
	ArpHrdEther uint16 = 0x0001
	ArpProIP    uint16 = 0x0800

	ArpHeaderSizeMin  uint8 = 8
	ArpIPEtherSizeMin uint8 = ArpHeaderSizeMin + 2*devices.EtherAddrLen + 2*ip.IPAddrLen

	ArpOpRequest uint16 = 1
	ArpOpReply   uint16 = 2
)

func ArpInit(done chan struct{}) error {
	go arpTimer(done)
	err := net.ProtocolRegister(&ArpProtocol{name: "arp0"})
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
	hrd uint16

	// protocol type
	pro uint16

	// hardware address length
	hln uint8

	// protocol address length
	pln uint8

	// opcode
	op uint16
}

// ArpEther is arp header for IPv4 and Ethernet
type ArpEther struct {

	// basic info for arp header
	ArpHeader

	// source hardware address
	sha [devices.EtherAddrLen]uint8

	// source protocol address
	spa ip.IPAddr

	// target hardware address
	tha [devices.EtherAddrLen]uint8

	// target protocol address
	tpa ip.IPAddr
}

func (ae ArpEther) String() string {
	return fmt.Sprintf(`
		hrd: %d
		pro: %d
		hln: %d
		pln: %d
		op: %d
		sha: %v
		spa: %s
		tha: %v
		tpa: %s
	`, ae.hrd, ae.pro, ae.hln, ae.pln, ae.op, ae.sha, ae.spa, ae.tha, ae.tpa)
}

// data2ArpHeader receives data and returns ARP header,the rest of data,error
func data2ArpHeader(data []byte) (ArpEther, []byte, error) {

	// only supports IPv4 and Ethernet address resolution
	if len(data) < int(ArpIPEtherSizeMin) {
		return ArpEther{}, nil, fmt.Errorf("data size is too small for arp header")
	}

	// read header
	var hdr ArpEther
	r := bytes.NewReader(data)
	binary.Read(r, binary.BigEndian, &hdr)

	// only receive IPv4 and Ethernet
	if hdr.hrd != ArpHrdEther || hdr.hln != devices.EtherAddrLen {
		return ArpEther{}, nil, fmt.Errorf("arp resolves only Ethernet address")
	}
	if hdr.pro != ArpProIP || hdr.pln != ip.IPAddrLen {
		return ArpEther{}, nil, fmt.Errorf("arp only supports IP address")
	}

	return hdr, data[ArpIPEtherSizeMin:], nil
}

/*
	Arp Protocol
*/

// ArpProtocol implements net.Protocol interface.
type ArpProtocol struct {
	name string
}

func (p *ArpProtocol) Name() string {
	return p.name
}

func (p *ArpProtocol) Type() net.ProtocolType {
	return net.ProtocolTypeArp
}

func (p *ArpProtocol) RxHandler(ch chan net.ProtocolBuffer, done chan struct{}) {
	var pb net.ProtocolBuffer
	var marge bool

	for {

		// check if finished or not
		select {
		case <-done:
			return
		default:
			marge = false
		}

		// receive data from device and transform it to header
		pb = <-ch
		hdr, _, err := data2ArpHeader(pb.Data)
		if err != nil {
			log.Printf("[E] %s", err.Error())
		}

		// update arp cache table
		mutex.Lock()
		if err := arpCacheUpdate(hdr.spa, hdr.sha[:]); err == nil {
			marge = true
		}
		mutex.Unlock()

		// search the IP interface of the device
		var ipIface *ip.IPIface
		var ok bool
		for _, iface := range pb.Dev.Interfaces() {
			if ipIface, ok = iface.(*ip.IPIface); ok {
				break
			}
		}
		if ipIface == nil || ipIface.Unicast != hdr.tpa {
			return // the data is to other host
		}

		// insert cache entry if entry is not updated before
		if !marge {
			mutex.Lock()
			arpCacheInsert(hdr.spa, hdr.sha[:])
			mutex.Unlock()
		}

		log.Printf("[D] dev=%s,arpheader=%s", pb.Dev.Name(), hdr)

		if hdr.op == ArpOpRequest {
			ArpReply(ipIface, hdr.sha, hdr.spa, hdr.tha[:]) // reply arp message
		}
	}
}

// ArpReply transmits ARP reply data to dst
func ArpReply(ipIface *ip.IPIface, tha [devices.EtherAddrLen]uint8, tpa ip.IPAddr, dst net.HardwareAddress) error {

	dev, ok := ipIface.Dev().(*devices.EthernetDevice)
	if !ok {
		return fmt.Errorf("arp only supports EthernetDevice")
	}

	// create arp header
	rep := ArpEther{
		ArpHeader: ArpHeader{
			hrd: ArpHrdEther,
			pro: ArpProIP,
			hln: devices.EtherAddrLen,
			pln: ip.IPAddrLen,
			op:  ArpOpReply,
		},
		sha: dev.Addr,
		spa: ipIface.Unicast,
		tha: tha,
		tpa: tpa,
	}

	// write data with bigendian
	w := bytes.NewBuffer(make([]byte, ArpIPEtherSizeMin))
	binary.Write(w, binary.BigEndian, rep)
	data := make([]byte, ArpIPEtherSizeMin)
	_, err := w.Write(data)
	if err != nil {
		return err
	}

	log.Printf("[D] ARP reply, dev=%s,arp header=%s", dev.Name(), rep)
	return net.DeviceOutput(dev, data, net.ProtocolTypeArp, dst)
}

// ArpResolve receives protocol address and returns hardware address
func ArpResolve(iface net.Interface, pa ip.IPAddr) (net.HardwareAddress, error) {

	// only supports IPv4 protocol
	ipIface, ok := iface.(*ip.IPIface)
	if !ok {
		return nil, fmt.Errorf("unsupported protocol address type")
	}

	// search cache table
	mutex.Lock()
	index, err := arpCacheSelect(pa)
	if err != nil {

		// if cache is not in the table, transmit arp request
		ArpRequest(ipIface, pa)

		mutex.Unlock()
		return nil, err
	}
	if caches[index].state == ArpCacheStateImcomplete {

		// if found cache is imcomplete,it might be a packet loss,so transmit arp request
		ArpRequest(ipIface, pa)

		mutex.Unlock()
		return nil, err
	}

	// get hardware address
	ha := caches[index].ha
	mutex.Unlock()
	return ha, nil
}

// ArpRequest receives interface and target IP address and transmits ARP request to the host(tpa)
func ArpRequest(ipIface *ip.IPIface, tpa ip.IPAddr) error {

	dev, ok := ipIface.Dev().(*devices.EthernetDevice)
	if !ok {
		return fmt.Errorf("arp only supports EthernetDevice")
	}

	// create arp header
	rep := ArpEther{
		ArpHeader: ArpHeader{
			hrd: ArpHrdEther,
			pro: ArpProIP,
			hln: devices.EtherAddrLen,
			pln: ip.IPAddrLen,
			op:  ArpOpRequest,
		},
		sha: dev.Addr,
		spa: ipIface.Unicast,
		tha: devices.EtherAddrAny,
		tpa: tpa,
	}

	// write data with bigendian
	w := bytes.NewBuffer(make([]byte, ArpIPEtherSizeMin))
	binary.Write(w, binary.BigEndian, rep)
	data := make([]byte, ArpIPEtherSizeMin)
	_, err := w.Write(data)
	if err != nil {
		return err
	}

	log.Printf("[D] ARP request, dev=%s,arp header=%s", dev.Name(), rep)
	return net.DeviceOutput(dev, data, net.ProtocolTypeArp, devices.EtherAddrBroadcast[:])
}
