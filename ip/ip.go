package ip

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/hedwig100/go-network/net"
	"github.com/hedwig100/go-network/utils"
)

type IPAddr uint32

const (
	IPVersionIPv4 = 4
	IPVersionIPv6 = 6

	IPHeaderSizeMin = 20

	IPAddrLen       uint8  = 4
	IPAddrAny       IPAddr = 0x00000000
	IPAddrBroadcast IPAddr = 0xffffffff
)

func IPInit(name string) (err error) {
	return
}

func (a IPAddr) String() string {
	b := uint32(a)
	return fmt.Sprintf("%d.%d,%d.%d", (b>>24)&0xff, (b>>16)&0xff, (b>>8)&0xff, b&0xff)
}

/*
	IP Header
*/
type IPHeader struct {

	// Version and Internet Header Length (4bit and 4bit)
	vhl uint8

	// Type Of Service
	tos uint8

	// Total Length
	tol uint16

	// Identification
	id uint16

	// flags and flagment offset (3bit and 13bit)
	flags uint16

	// Time To Live
	ttl uint8

	// protocol Type
	protocolType IPProtocolType

	// checksum
	checksum uint16

	// source IP address and destination IP address
	src IPAddr
	dst IPAddr
}

func (h *IPHeader) String() string {
	return fmt.Sprintf(`
		version: %d,
		header length: %d,
		tos: %d,
		id: %d,
		flags: %x,
		ttl: %d,
		protocolType: %s,
		checksum: %x,
		src: %s,
		dst: %s,
	`, h.vhl>>4, h.vhl&0xf, h.tos, h.id, h.flags, h.ttl, h.protocolType, h.checksum, h.src, h.dst)
}

func data2IPHeader(b []byte) (ipHdr IPHeader, data []byte, err error) {

	if len(b) < IPHeaderSizeMin {
		err = fmt.Errorf("data size is too small")
		return
	}

	r := bytes.NewReader(b)
	binary.Read(r, binary.BigEndian, &ipHdr)
	if (ipHdr.vhl >> 4) != IPVersionIPv4 {
		err = fmt.Errorf("version is not valid")
		return
	}

	hlen := ipHdr.vhl & 0x0f
	if uint8(len(b)) < hlen {
		err = fmt.Errorf("data length is smaller than IHL")
		return
	}

	if uint16(len(b)) < ipHdr.tol {
		err = fmt.Errorf("data length is smaller than Total Length")
		return
	}

	if utils.CheckSum(b[:hlen]) != 0 {
		err = fmt.Errorf("checksum is not valid")
		return
	}

	log.Printf("[D] ip header is received")
	data = b[hlen:]
	return
}

var id uint16 = 0

// generateId() generates id for IP header
func generateId() uint16 {
	id++
	return id
}

// IpHead2byte encodes IP header data to a strings of bytes
func IPHead2byte(protocol IPProtocolType, src IPAddr, dst IPAddr, data []byte, flags uint16) ([]byte, error) {

	// ip header
	hdr := IPHeader{
		vhl:          (IPVersionIPv4<<4 | IPHeaderSizeMin>>2),
		tol:          uint16(IPHeaderSizeMin + len(data)),
		id:           generateId(),
		flags:        flags,
		ttl:          0xff,
		protocolType: protocol,
		checksum:     0,
		src:          src,
		dst:          dst,
	}

	// encoding BigEndian
	rdr := bytes.NewReader(make([]byte, IPHeaderSizeMin))
	binary.Read(rdr, binary.BigEndian, hdr)
	buf := make([]byte, IPHeaderSizeMin)
	_, err := rdr.Read(buf)
	if err != nil {
		return nil, err
	}

	// caluculate checksum
	chs := utils.CheckSum(buf)
	copy(buf[10:12], utils.Hton16(chs))
	return buf, nil
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

func (p *IPProtocol) Type() net.ProtocolType {
	return net.ProtocolTypeIP
}

func (p *IPProtocol) TxHandler(protocol IPProtocolType, data []byte, src IPAddr, dst IPAddr) error {

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
	for _, iface := range net.Interfaces {
		ipIface, ok := iface.(*IPIface)
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

	// get IP header
	hdr, err := IPHead2byte(protocol, src, dst, data, 0)
	if err != nil {
		return err
	}
	data = append(hdr, data...) // more efficient implementation?

	// TODO:ARP
	// use route.nexthop here (in the future)
	// transmit data from device
	var hwadddr net.HardwareAddress
	err = ipIface.dev.Transmit(data, net.ProtocolTypeIP, hwadddr)

	return err
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
		ipHdr, data, err := data2IPHeader(pb.Data)
		if err != nil {
			log.Printf("[E] IP RxHandler %s", err.Error())
			continue
		}

		if ipHdr.flags&0x2000 > 0 || ipHdr.flags&0x1fff > 0 {
			log.Printf("[E] IP RxHandler does not support fragments")
			continue
		}

		// search the interface whose address matches the header's one
		var ipIface *IPIface
		for _, iface := range pb.Dev.Interfaces() {
			ipIface, ok := iface.(*IPIface)
			if ok && (ipIface.Unicast == ipHdr.dst || ipIface.broadcast == IPAddrBroadcast || ipIface.broadcast == ipHdr.dst) {
				break
			}
		}
		if ipIface == nil {
			return // the packet is to other host
		}
		log.Printf("[D] IP header=%v,iface=%v,protocol=%s", ipHdr, ipIface, ipHdr.protocolType)

		// search the protocol whose type is the same as the header's one
		for _, proto := range IPProtocols {
			if proto.Type() == ipHdr.protocolType {
				proto.RxHandler(data, ipHdr.src, ipHdr.dst, ipIface)
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
	dev net.Device

	// unicast address ex) 192.0.0.1
	Unicast IPAddr

	// netmask ex) 255.255.255.0
	netmask IPAddr

	// broadcast address for the subnet
	broadcast IPAddr
}

// NewIPIface returns IPIface whose address is unicastStr
func NewIPIface(unicastStr string, netmaskStr string) (iface *IPIface, err error) {

	unicast, err := str2IPAddr(unicastStr)
	if err != nil {
		return
	}

	netmask, err := str2IPAddr(unicastStr)
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

func (i *IPIface) Dev() net.Device {
	return i.dev
}

func (i *IPIface) SetDev(dev net.Device) {
	i.dev = dev
}

func (i *IPIface) Family() int {
	return net.NetIfaceFamilyIP
}

// IPIfaceRegister registers ipIface to dev
func IPIfaceRegister(dev net.Device, ipIface *IPIface) {
	net.IfaceRegister(dev, ipIface)

	// register subnet's routing information to routing table
	// this information is used when data is sent to the subnet's host
	IPRouteAdd(ipIface.Unicast&ipIface.netmask, ipIface.netmask, IPAddrAny, ipIface)
}

// SearchIpIface searches an interface which has the IP address
func SearchIPIface(addr IPAddr) (*IPIface, error) {
	for _, iface := range net.Interfaces {
		iface, ok := iface.(*IPIface)
		if ok && iface.Unicast == addr {
			return iface, nil
		}
	}
	return nil, fmt.Errorf("not found IP interface(addr=%d)", addr)
}

// ex) "127.0.0.1" -> 01111111 00000000 00000000 00000001
func str2IPAddr(str string) (uint32, error) {
	strs := strings.Split(str, ".")
	var b uint32
	for i, s := range strs {
		n, err := strconv.Atoi(s)
		if err != nil {
			return 0, err
		}
		b |= uint32((n & 0x000f) << (24 - 8*i))
	}
	return b, nil
}
