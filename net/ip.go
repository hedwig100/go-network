package net

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
)

const (
	IPVersionIPv4 = 4
	IPVersionIPv6 = 6

	IPHeaderSizeMin = 20

	IPAddrLen       uint8  = 4
	IPAddrAny       IPAddr = 0x00000000
	IPAddrBroadcast IPAddr = 0xffffffff
)

func IPInit() error {
	err := ProtocolRegister(&IPProtocol{name: "ip0"})
	return err
}

/*
	IP address
*/

// IPAddr is IP address
type IPAddr uint32

func (a IPAddr) String() string {
	b := uint32(a)
	return fmt.Sprintf("%d.%d.%d.%d", (b>>24)&0xff, (b>>16)&0xff, (b>>8)&0xff, b&0xff)
}

/*
	IP Header
*/

// IPHeader is header for IP packet.
type IPHeader struct {

	// Version and Internet Header Length (4bit and 4bit)
	Vhl uint8

	// Type Of Service
	Tos uint8

	// Total Length
	Tol uint16

	// Identification
	Id uint16

	// flags and flagment offset (3bit and 13bit)
	Flags uint16

	// Time To Live
	Ttl uint8

	// protocol Type
	ProtocolType IPProtocolType

	// checksum
	Checksum uint16

	// source IP address and destination IP address
	Src IPAddr
	Dst IPAddr
}

func (h IPHeader) String() string {
	return fmt.Sprintf(`
		Version: %d,
		Header Length: %d,
		Total Length: %d,
		Tos: %d,
		Id: %d,
		Flags: %x,
		Fragment Offset: %d,
		TTL: %d,
		ProtocolType: %s,
		Checksum: %x,
		Src: %s,
		Dst: %s,
	`, h.Vhl>>4, h.Vhl&0xf<<2, h.Tol, h.Tos, h.Id, h.Flags>>13, h.Flags&0x1fff, h.Ttl, h.ProtocolType, h.Checksum, h.Src, h.Dst)
}

// data2IPHeader transforms byte strings to IP Header and the rest of the data
func data2headerIP(data []byte) (IPHeader, []byte, error) {

	if len(data) < IPHeaderSizeMin {
		return IPHeader{}, nil, fmt.Errorf("data size is too small")
	}

	// read header in bigEndian
	var ipHdr IPHeader
	r := bytes.NewReader(data)
	err := binary.Read(r, binary.BigEndian, &ipHdr)
	if err != nil {
		return IPHeader{}, nil, err
	}

	// check if the packet is iPv4
	if (ipHdr.Vhl >> 4) != IPVersionIPv4 {
		return IPHeader{}, nil, err
	}

	// check header length and total length
	hlen := (ipHdr.Vhl & 0xf) << 2
	if uint8(len(data)) < hlen {
		return IPHeader{}, nil, fmt.Errorf("data length is smaller than IHL")
	}
	if uint16(len(data)) < ipHdr.Tol {
		return IPHeader{}, nil, fmt.Errorf("data length is smaller than Total Length")
	}

	// calculate checksum
	chksum := CheckSum(data[:hlen])
	if chksum != 0 && chksum != 0xffff { // 0 or -0
		return IPHeader{}, nil, fmt.Errorf("checksum is not valid")
	}

	return ipHdr, data[hlen:], nil
}

func header2dataIP(hdr *IPHeader, payload []byte) ([]byte, error) {

	// write header in bigEndian
	var w bytes.Buffer
	err := binary.Write(&w, binary.BigEndian, hdr)
	if err != nil {
		return nil, err
	}

	// write payload as it is
	_, err = w.Write(payload)
	if err != nil {
		return nil, err
	}

	// caluculate checksum
	buf := w.Bytes()
	chksum := CheckSum(buf[:IPHeaderSizeMin])
	copy(buf[10:12], Hton16(chksum))

	// set checksum in the header (for debug)
	hdr.Checksum = chksum
	return buf, nil
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
type IPProtocol struct {
	name string
}

func (p *IPProtocol) Name() string {
	return p.name
}

func (p *IPProtocol) Type() ProtocolType {
	return ProtocolTypeIP
}

func TxHandlerIP(protocol IPProtocolType, data []byte, src IPAddr, dst IPAddr) error {

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
	if src != IPAddrAny && src != route.ipIface.Unicast {
		return fmt.Errorf("unable to output with specified source address,addr=%s", src)
	}

	// search the interface whose address matches src
	var ipIface *IPIface
	var ok bool
	for _, iface := range Interfaces {
		ipIface, ok = iface.(*IPIface)
		if ok && src == ipIface.Unicast {
			break
		}
	}
	if ipIface == nil {
		return fmt.Errorf("IP interface whose address is %v is not found", src)
	}

	// check if dst is broadcast address of IP interface broadcast address
	if dst != IPAddrBroadcast && (uint32(dst)|uint32(ipIface.broadcast)) != uint32(ipIface.broadcast) {
		return fmt.Errorf("dst(%v) IP address cannot be reachable(broadcast=%v)", dst, ipIface.broadcast)
	}

	// does not support fragmentation
	if int(ipIface.dev.MTU()) < IPHeaderSizeMin+len(data) {
		return fmt.Errorf("dst(%v) IP address cannot be reachable(broadcast=%v)", dst, ipIface.broadcast)
	}

	// transform IP header to byte strings
	hdr := IPHeader{
		Vhl:          (IPVersionIPv4<<4 | IPHeaderSizeMin>>2),
		Tol:          uint16(IPHeaderSizeMin + len(data)),
		Id:           generateId(),
		Flags:        0,
		Ttl:          0xff,
		ProtocolType: protocol,
		Checksum:     0,
		Src:          src,
		Dst:          dst,
	}
	data, err = header2dataIP(&hdr, data)
	if err != nil {
		return err
	}

	// transmit data from the device
	var hwaddr HardwareAddress
	if ipIface.dev.Flags()&NetDeviceFlagNeedARP > 0 { // check if arp is necessary
		if dst == ipIface.broadcast || dst == IPAddrBroadcast {
			hwaddr = EtherAddrBroadcast // TODO: not only ethernet
		} else {
			hwaddr, err = ArpResolve(ipIface, dst)
			if err != nil {
				return err
			}
		}
	}

	log.Printf("[D] IP TxHandler: iface=%d,dev=%s,header=%s", ipIface.Family(), ipIface.dev.Name(), hdr)
	return ipIface.dev.Transmit(data, ProtocolTypeIP, hwaddr)
}

func (p *IPProtocol) RxHandler(ch chan ProtocolBuffer, done chan struct{}) {
	var pb ProtocolBuffer

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
			log.Printf("[E] IP RxHandler: %s", err.Error())
			continue
		}

		if ipHdr.Flags&0x2000 > 0 || ipHdr.Flags&0x1fff > 0 {
			log.Printf("[E] IP RxHandler: does not support fragments")
			continue
		}

		// search the interface whose address matches the header's one
		var ipIface *IPIface
		var ok bool
		for _, iface := range pb.Dev.Interfaces() {
			ipIface, ok = iface.(*IPIface)
			if ok && (ipIface.Unicast == ipHdr.Dst || ipIface.broadcast == IPAddrBroadcast || ipIface.broadcast == ipHdr.Dst) {
				break
			}
		}
		if ipIface == nil {
			log.Printf("[D] IP RxHandler: packet is to other host")
			continue // the packet is to other host
		}
		log.Printf("[D] IP RxHandler: iface=%s,protocol=%s,header=%v", ipIface.Unicast, ipHdr.ProtocolType, ipHdr)

		// search the protocol whose type is the same as the header's one
		for _, proto := range IPProtocols {
			if proto.Type() == ipHdr.ProtocolType {
				err = proto.RxHandler(payload, ipHdr.Src, ipHdr.Dst, ipIface)
				if err != nil {
					log.Printf("[E] IP RxHanlder: %s", err.Error())
				}
			}
		}
	}
}

/*
	IP logical Interface
*/

// IPIface is IP interface
// *IPIface implements net.Interface
type IPIface struct {

	// device of the interface
	dev Device

	// unicast address ex) 192.0.0.1
	Unicast IPAddr

	// netmask ex) 255.255.255.0
	netmask IPAddr

	// broadcast address for the subnet
	broadcast IPAddr
}

// NewIPIface returns IPIface whose address is unicastStr
func NewIPIface(unicastStr string, netmaskStr string) (iface *IPIface, err error) {

	unicast, err := Str2IPAddr(unicastStr)
	if err != nil {
		return
	}

	netmask, err := Str2IPAddr(netmaskStr)
	if err != nil {
		return
	}

	iface = &IPIface{
		Unicast:   IPAddr(unicast),
		netmask:   IPAddr(netmask),
		broadcast: IPAddr(unicast | ^netmask),
	}
	return
}

func (i *IPIface) Dev() Device {
	return i.dev
}

func (i *IPIface) SetDev(dev Device) {
	i.dev = dev
}

func (i *IPIface) Family() int {
	return NetIfaceFamilyIP
}

// IPIfaceRegister registers ipIface to dev
func IPIfaceRegister(dev Device, ipIface *IPIface) {
	IfaceRegister(dev, ipIface)

	// register subnet's routing information to routing table
	// this information is used when data is sent to the subnet's host
	IPRouteAdd(ipIface.Unicast&ipIface.netmask, ipIface.netmask, IPAddrAny, ipIface)
}

// SearchIpIface searches an interface which has the IP address
func SearchIPIface(addr IPAddr) (*IPIface, error) {
	for _, iface := range Interfaces {
		iface, ok := iface.(*IPIface)
		if ok && iface.Unicast == addr {
			return iface, nil
		}
	}
	return nil, fmt.Errorf("not found IP interface(addr=%d)", addr)
}
