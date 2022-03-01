package udp

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"

	"github.com/hedwig100/go-network/pkg/ip"
	"github.com/hedwig100/go-network/pkg/utils"
)

// UDPInit prepare the UDP protocol.
func UDPInit() error {
	return ip.IPProtocolRegister(&UDPProtocol{})
}

/*
	UDP endpoint
*/

const (
	UDPPortMin uint16 = 49152
	UDPPortMax uint16 = 65535
)

// UDPEndpoint is IP address and port number combination
type UDPEndpoint struct {

	// IP address
	Address ip.IPAddr

	// port number
	Port uint16
}

func (e UDPEndpoint) String() string {
	return fmt.Sprintf("%s:%d", e.Address, e.Port)
}

// Str2UDPEndpoint encodes str to UDPEndpoint
// ex) str="8.8.8.8:80"
func Str2UDPEndpoint(str string) (UDPEndpoint, error) {
	tmp := strings.Split(str, ":")
	if len(tmp) != 2 {
		return UDPEndpoint{}, fmt.Errorf("str is not correect")
	}
	addr, err := ip.Str2IPAddr(tmp[0])
	if err != nil {
		return UDPEndpoint{}, err
	}
	port, err := strconv.Atoi(tmp[1])
	if err != nil {
		return UDPEndpoint{}, err
	}
	return UDPEndpoint{
		Address: ip.IPAddr(addr),
		Port:    uint16(port),
	}, nil
}

/*
	UDP Header
*/

const (
	UDPHeaderSize       = 8
	UDPPseudoHeaderSize = 12 // only supports IPv4 now
)

// UDPHeader is header for UDP.
type UDPHeader struct {

	// Source port number
	Src uint16

	// Destination port number
	Dst uint16

	// packet length
	Len uint16

	// checksum
	Checksum uint16
}

func (h UDPHeader) String() string {
	return fmt.Sprintf(`
		Src: %d,
		Dst: %d,
		Len: %d,
		Checksum: %x,
	`, h.Src, h.Dst, h.Len, h.Checksum)
}

// UDPPseudoHeader is used for caluculating checksum
type UDPPseudoHeader struct {

	// source IP address
	Src ip.IPAddr

	// destination IP address
	Dst ip.IPAddr

	// padding, always 0
	Pad uint8

	// protocol type, always 17(UDP)
	Type ip.IPProtocolType

	// UDP packet length
	Len uint16
}

// data2headerUDP transforms data to UDP header
// src,dst is used for caluculating checksum
func data2headerUDP(data []byte, src ip.IPAddr, dst ip.IPAddr) (UDPHeader, []byte, error) {

	if len(data) < UDPHeaderSize {
		return UDPHeader{}, nil, fmt.Errorf("data size is too small for udp header")
	}

	// read header in bigEndian
	var hdr UDPHeader
	r := bytes.NewReader(data)
	err := binary.Read(r, binary.BigEndian, &hdr)
	if err != nil {
		return UDPHeader{}, nil, err
	}

	// check if length is correct
	if int(hdr.Len) != len(data) {
		return UDPHeader{}, nil, fmt.Errorf("data length is not the same as that written in header")
	}

	// checksum not supported to other host
	if hdr.Checksum == 0 {
		return hdr, data[UDPHeaderSize:], nil
	}

	// caluculate checksum
	pseudoHdr := UDPPseudoHeader{
		Src:  src,
		Dst:  dst,
		Type: ip.IPProtocolUDP,
		Len:  hdr.Len,
	}
	var w bytes.Buffer
	binary.Write(&w, binary.BigEndian, pseudoHdr)
	chksum := utils.CheckSum(w.Bytes(), 0)
	chksum = utils.CheckSum(data, uint32(^chksum))
	if chksum != 0 && chksum != 0xffff {
		return UDPHeader{}, nil, fmt.Errorf("checksum error (UDP)")
	}

	return hdr, data[UDPHeaderSize:], nil
}

func header2dataUDP(hdr *UDPHeader, payload []byte, src ip.IPAddr, dst ip.IPAddr) ([]byte, error) {

	// pseudo header for caluculating checksum afterwards
	pseudoHdr := UDPPseudoHeader{
		Src:  src,
		Dst:  dst,
		Type: ip.IPProtocolUDP,
		Len:  uint16(UDPHeaderSize + len(payload)),
	}

	// write header in bigEndian
	var w bytes.Buffer
	err := binary.Write(&w, binary.BigEndian, pseudoHdr)
	if err != nil {
		return nil, err
	}
	err = binary.Write(&w, binary.BigEndian, hdr)
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
	chksum := utils.CheckSum(buf, 0)
	copy(buf[18:20], utils.Hton16(chksum))

	// set checksum in the header (for debug)
	hdr.Checksum = chksum
	return buf[UDPPseudoHeaderSize:], nil
}

/*
	UDP Protocol
*/
// UDPProtocol is struct for UDP protocol handler.
// This implements IPUpperProtocol interface.
type UDPProtocol struct{}

func (p *UDPProtocol) Type() ip.IPProtocolType {
	return ip.IPProtocolUDP
}

func (p *UDPProtocol) RxHandler(data []byte, src ip.IPAddr, dst ip.IPAddr, ipIface *ip.IPIface) error {
	hdr, payload, err := data2headerUDP(data, src, dst)
	if err != nil {
		return err
	}
	log.Printf("[D] UDP rxHandler: src=%s:%d,dst=%s:%d,iface=%s,udp header=%s,payload=%v", src, hdr.Src, dst, hdr.Dst, ipIface.Family(), hdr, payload)

	// search udp pcb whose address is dst
	udpMutex.Lock()
	defer udpMutex.Unlock()
	pcb := udpPCBSelect(dst, hdr.Dst)
	if pcb == nil {
		return fmt.Errorf("destination UDP protocol control block not found")
	}

	pcb.rxQueue <- udpBuffer{
		foreign: UDPEndpoint{
			Address: src,
			Port:    hdr.Src,
		},
		data: payload,
	}
	return nil
}

// TxHandlerUDP transmits UDP datagram to the other host.
func TxHandlerUDP(src UDPEndpoint, dst UDPEndpoint, data []byte) error {

	if len(data)+UDPHeaderSize > ip.IPPayloadSizeMax {
		return fmt.Errorf("data size is too large for UDP payload")
	}

	// transform UDP header to byte strings
	hdr := UDPHeader{
		Src: src.Port,
		Dst: dst.Port,
		Len: uint16(UDPHeaderSize + len(data)),
	}
	data, err := header2dataUDP(&hdr, data, src.Address, dst.Address)
	if err != nil {
		return err
	}

	log.Printf("[D] UDP TxHandler: src=%s,dst=%s,udp header=%s", src, dst, hdr)
	return ip.TxHandlerIP(ip.IPProtocolUDP, data, src.Address, dst.Address)
}

/*
	Protocol Control Block (for socket API)
*/

const (
	udpPCBStateFree    = 0
	udpPCBStateOpen    = 1
	udpPCBStateClosing = 2

	udpPCBBufSize = 100
)

var (
	udpMutex sync.Mutex
	pcbs     []*UDPpcb
)

// UDPpcb is protocol control block for UDP
type UDPpcb struct {

	// pcb state
	state int

	// our UDP endpoint
	local UDPEndpoint

	// receive queue
	rxQueue chan udpBuffer
}

// udpBuffer is
type udpBuffer struct {

	// UDP endpoint of the source
	foreign UDPEndpoint

	// data sent to us
	data []byte
}

func udpPCBSelect(address ip.IPAddr, port uint16) *UDPpcb {
	for _, p := range pcbs {
		if p.local.Address == address && p.local.Port == port {
			return p
		}
	}
	return nil
}

func OpenUDP() *UDPpcb {
	pcb := &UDPpcb{
		state: udpPCBStateOpen,
		local: UDPEndpoint{
			Address: ip.IPAddrAny,
		},
		rxQueue: make(chan udpBuffer, udpPCBBufSize),
	}
	udpMutex.Lock()
	pcbs = append(pcbs, pcb)
	udpMutex.Unlock()
	return pcb
}

func Close(pcb *UDPpcb) error {

	index := -1
	udpMutex.Lock()
	defer udpMutex.Unlock()
	for i, p := range pcbs {
		if p == pcb {
			index = i
			break
		}
	}
	if index < 0 {
		return fmt.Errorf("pcb not found")
	}

	pcbs = append(pcbs[:index], pcbs[index+1:]...)
	return nil
}

func (pcb *UDPpcb) Bind(local UDPEndpoint) error {

	// check if the same address has not been bound
	udpMutex.Lock()
	defer udpMutex.Unlock()
	for _, p := range pcbs {
		if p.local == local {
			return fmt.Errorf("local address(%s) is already binded", local)
		}
	}
	pcb.local = local
	log.Printf("[I] bound address local=%s", local)
	return nil
}

func (pcb *UDPpcb) Send(data []byte, dst UDPEndpoint) error {

	local := pcb.local

	if local.Address == ip.IPAddrAny {
		route, err := ip.LookupTable(dst.Address)
		if err != nil {
			return err
		}
		local.Address = route.IpIface.Unicast
	}

	if local.Port == 0 { // zero value of Port (uint16)
		for p := UDPPortMin; p <= UDPPortMax; p++ {
			if udpPCBSelect(local.Address, p) != nil {
				local.Port = p
				log.Printf("[D] registered UDP :address=%s,port=%d", local.Address, local.Port)
				break
			}
		}
		if local.Port == 0 {
			return fmt.Errorf("there is no port number to assign")
		}
	}

	return TxHandlerUDP(local, dst, data)
}

// Listen listens data and write data to 'data'. if 'block' is false, there is no blocking I/O.
// This function returns data size,data,and source UDP endpoint.
func (pcb *UDPpcb) Listen(block bool) (int, []byte, UDPEndpoint) {

	if block {
		buf := <-pcb.rxQueue
		return len(buf.data), buf.data, buf.foreign
	}

	// no blocking
	select {
	case buf := <-pcb.rxQueue:
		return len(buf.data), buf.data, buf.foreign
	default:
		return 0, []byte{}, UDPEndpoint{}
	}
}
