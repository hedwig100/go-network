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

type IpAddr uint32

const (
	IpVersionIPv4 = 4
	IpVersionIPv6 = 6

	IpHeaderSizeMin = 20

	IpAddrAny       IpAddr = 0x00000000
	IpAddrBroadcast IpAddr = 0xffffffff
)

func IpInit(name string) (err error) {
	return
}

/*
	IP Header
*/
type IpHeader struct {

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
	protocolType IpProtocolType

	// checksum
	checksum uint16

	// source IP address and destination IP address
	src IpAddr
	dst IpAddr
}

func (h *IpHeader) String() string {
	return ""
}

func data2IpHeader(b []byte) (ipHdr IpHeader, data []byte, err error) {

	if len(b) < IpHeaderSizeMin {
		err = fmt.Errorf("data size is too small")
		return
	}

	r := bytes.NewReader(b)
	binary.Read(r, binary.BigEndian, &ipHdr)
	if (ipHdr.vhl >> 4) != IpVersionIPv4 {
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
func IpHead2byte(protocol IpProtocolType, src IpAddr, dst IpAddr, data []byte, flags uint16) ([]byte, error) {

	// ip header
	hdr := IpHeader{
		vhl:          (IpVersionIPv4<<4 | IpHeaderSizeMin>>2),
		tol:          uint16(IpHeaderSizeMin + len(data)),
		id:           generateId(),
		flags:        flags,
		ttl:          0xff,
		protocolType: protocol,
		checksum:     0,
		src:          src,
		dst:          dst,
	}

	// encoding BigEndian
	rdr := bytes.NewReader(make([]byte, IpHeaderSizeMin))
	binary.Read(rdr, binary.BigEndian, hdr)
	buf := make([]byte, IpHeaderSizeMin)
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
type IpProtocol struct {
	name string
}

func (p *IpProtocol) Name() string {
	return p.name
}

func (p *IpProtocol) Type() net.ProtocolType {
	return net.ProtocolTypeIP
}

func (p *IpProtocol) TxHandler(protocol IpProtocolType, data []byte, src IpAddr, dst IpAddr) error {

	// TODO:implement IP rooting
	if src == IpAddrAny {
		log.Printf("ip routing is not implemented yet")
		return nil
	}

	// search the interface whose address matches src
	var ipIface *IpIface
	for _, iface := range net.Interfaces {
		ipIface, ok := iface.(*IpIface)
		if ok && src == ipIface.unicast {
			break
		}
	}
	if ipIface == nil {
		return fmt.Errorf("IP interface whose address is %v is not found", src)
	}

	// check if dst is broadcast address of IP interface broadcast address
	if dst != IpAddrBroadcast && (uint32(dst)|uint32(ipIface.broadcast)) != uint32(ipIface.broadcast) {
		return fmt.Errorf("dst(%v) IP address cannot be reachable(broadcast=%v)", dst, ipIface.broadcast)
	}

	// does not support fragmentation
	if int(ipIface.dev.MTU()) < IpHeaderSizeMin+len(data) {
		return fmt.Errorf("dst(%v) IP address cannot be reachable(broadcast=%v)", dst, ipIface.broadcast)
	}

	// get IP header
	hdr, err := IpHead2byte(protocol, src, dst, data, 0)
	if err != nil {
		return err
	}
	data = append(hdr, data...) // more efficient implementation?

	// TODO:ARP
	// transmit data from device
	var hwadddr net.HardwareAddress
	err = ipIface.dev.Transmit(data, net.ProtocolTypeIP, hwadddr)

	return err
}

func (p *IpProtocol) RxHandler(ch chan net.ProtocolBuffer, done chan struct{}) {
	var pb net.ProtocolBuffer

	for {

		// 終了したかどうか確認
		// check if finished or not
		select {
		case <-done:
			return
		default:
		}

		// receive data from device
		pb = <-ch

		// extract the header from the beginning of the data
		ipHdr, data, err := data2IpHeader(pb.Data)
		if err != nil {
			log.Printf("[E] IP RxHandler %s", err.Error())
			continue
		}

		if ipHdr.flags&0x2000 > 0 || ipHdr.flags&0x1fff > 0 {
			log.Printf("[E] IP RxHandler does not support fragments")
			continue
		}

		// search the interface whose address matches the header's one
		var ipIface *IpIface
		for _, iface := range pb.Dev.Interfaces() {
			ipIface, ok := iface.(*IpIface)
			if ok && (ipIface.unicast == ipHdr.dst || ipIface.broadcast == IpAddrBroadcast || ipIface.broadcast == ipHdr.dst) {
				break
			}
		}
		if ipIface == nil {
			return // the packet is to other host
		}
		log.Printf("[D] IP header=%v,iface=%v,protocol=%s", ipHdr, ipIface, ipHdr.protocolType)

		// search the protocol whose type is the same as the header's one
		for _, proto := range IpProtocols {
			if proto.Type() == ipHdr.protocolType {
				proto.RxHandler(data, ipHdr.src, ipHdr.dst, ipIface)
			}
		}
	}
}

/*
	IP logical Interface
*/
type IpIface struct {
	dev       net.Device
	unicast   IpAddr
	netmask   IpAddr
	broadcast IpAddr
}

func NewIpIface(unicastStr string, netmaskStr string) (iface *IpIface, err error) {

	unicast, err := str2IpAddr(unicastStr)
	if err != nil {
		return
	}

	netmask, err := str2IpAddr(unicastStr)
	if err != nil {
		return
	}

	iface = &IpIface{
		unicast:   IpAddr(unicast),
		netmask:   IpAddr(netmask),
		broadcast: IpAddr(unicast | ^netmask),
	}
	return
}

func (i *IpIface) Dev() net.Device {
	return i.dev
}

func (i *IpIface) SetDev(dev net.Device) {
	i.dev = dev
}

func (i *IpIface) Family() int {
	return net.NetIfaceFamilyIP
}

// あるIPアドレスを持つインタフェースを探す
// search an interface which has the IP address
func SearchIpIface(addr IpAddr) (*IpIface, error) {
	for _, iface := range net.Interfaces {
		iface, ok := iface.(*IpIface)
		if ok && iface.unicast == addr {
			return iface, nil
		}
	}
	return nil, fmt.Errorf("not found IP interface(addr=%d)", addr)
}

// ex) "127.0.0.1" -> 01111111 00000000 00000000 00000001
func str2IpAddr(str string) (uint32, error) {
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
