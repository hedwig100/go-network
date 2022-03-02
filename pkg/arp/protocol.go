package arp

import (
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

	HeaderSizeMin uint8 = 8
	ArpEtherSize  uint8 = HeaderSizeMin + 2*device.EtherAddrLen + 2*ip.AddrLen

	arpOpRequest uint16 = 1
	arpOpReply   uint16 = 2
)

// Init prepare the ARP protocol.
func Init(done chan struct{}) error {
	go arpTimer(done)
	err := net.ProtoRegister(&Proto{})
	if err != nil {
		return err
	}
	return nil
}

// Proto implements net.Protocol interface.
type Proto struct{}

func (p *Proto) Type() net.ProtoType {
	return net.ProtoTypeArp
}

func (p *Proto) RxHandler(ch chan net.ProtoBuffer, done chan struct{}) {
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
		hdr, err := data2header(pb.Data)
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
			err = Reply(ipIface, hdr.Sha, hdr.Spa, hdr.Sha) // reply arp message
			if err != nil {
				log.Printf("[E] ARP rxHandler: %s", err.Error())
			}
		}
	}
}

// Reply transmits ARP reply data to dst
func Reply(ipIface *ip.Iface, tha device.EtherAddr, tpa ip.Addr, dst device.EtherAddr) error {

	dev, ok := ipIface.Dev().(*device.Ether)
	if !ok {
		return fmt.Errorf("arp only supports EthernetDevice")
	}

	// create arp header
	rep := ArpEther{
		Header: Header{
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

	data, err := header2data(rep)
	if err != nil {
		return err
	}

	log.Printf("[D] ARP TxHandler(reply): dev=%s,arp header=%s", dev.Name(), rep)
	return net.DeviceOutput(dev, data, net.ProtoTypeArp, dst)
}

// Resolve receives protocol address and returns hardware address
func Resolve(iface net.Interface, pa ip.Addr) (net.HardwareAddr, error) {

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
		Request(ipIface, pa)
		mutex.Unlock()

		return nil, err
	}

	// cache found but imcomplete request
	if caches[index].state == arpCacheStateImcomplete {

		// if found cache is imcomplete,it might be a packet loss,so transmit arp request
		Request(ipIface, pa)
		mutex.Unlock()

		return nil, fmt.Errorf("cache state is imcomplete")
	}

	// cache found and get hardware address
	ha := caches[index].ha
	mutex.Unlock()
	return ha, nil
}

// Request receives interface and target IP address and transmits ARP request to the host(tpa)
func Request(ipIface *ip.Iface, tpa ip.Addr) error {

	dev, ok := ipIface.Dev().(*device.Ether)
	if !ok {
		return fmt.Errorf("arp only supports EthernetDevice")
	}

	// create arp header
	rep := ArpEther{
		Header: Header{
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

	data, err := header2data(rep)
	if err != nil {
		return err
	}

	log.Printf("[D] ARP TxHandler(request): dev=%s,arp header=%s", dev.Name(), rep)
	return net.DeviceOutput(dev, data, net.ProtoTypeArp, device.EtherAddrBroadcast)
}
