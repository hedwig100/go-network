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

// Init prepare the UDP protocol.
func Init() error {
	return ip.ProtoRegister(&Protocol{})
}

/*
	UDP endpoint
*/

const (
	PortMin uint16 = 49152
	PortMax uint16 = 65535
)

// Endpoint is IP address and port number combination
type Endpoint struct {

	// IP address
	Addr ip.Addr

	// port number
	Port uint16
}

func (e Endpoint) String() string {
	return fmt.Sprintf("%s:%d", e.Addr, e.Port)
}

// Str2Endpoint encodes str to Endpoint
// ex) str="8.8.8.8:80"
func Str2Endpoint(str string) (Endpoint, error) {
	tmp := strings.Split(str, ":")
	if len(tmp) != 2 {
		return Endpoint{}, fmt.Errorf("str is not correect")
	}
	addr, err := ip.Str2Addr(tmp[0])
	if err != nil {
		return Endpoint{}, err
	}
	port, err := strconv.Atoi(tmp[1])
	if err != nil {
		return Endpoint{}, err
	}
	return Endpoint{
		Addr: ip.Addr(addr),
		Port: uint16(port),
	}, nil
}

/*
	UDP Header
*/

const (
	HeaderSize       = 8
	PseudoHeaderSize = 12 // only supports IPv4 now
)

// Header is header for UDP.
type Header struct {

	// Source port number
	Src uint16

	// Destination port number
	Dst uint16

	// packet length
	Len uint16

	// checksum
	Checksum uint16
}

func (h Header) String() string {
	return fmt.Sprintf(`
		Src: %d,
		Dst: %d,
		Len: %d,
		Checksum: %x,
	`, h.Src, h.Dst, h.Len, h.Checksum)
}

// PseudoHeader is used for caluculating checksum
type PseudoHeader struct {

	// source IP address
	Src ip.Addr

	// destination IP address
	Dst ip.Addr

	// padding, always 0
	Pad uint8

	// protocol type, always 17(UDP)
	Type ip.ProtoType

	// UDP packet length
	Len uint16
}

// data2header transforms data to UDP header
// src,dst is used for caluculating checksum
func data2header(data []byte, src ip.Addr, dst ip.Addr) (Header, []byte, error) {

	if len(data) < HeaderSize {
		return Header{}, nil, fmt.Errorf("data size is too small for udp header")
	}

	// read header in bigEndian
	var hdr Header
	r := bytes.NewReader(data)
	err := binary.Read(r, binary.BigEndian, &hdr)
	if err != nil {
		return Header{}, nil, err
	}

	// check if length is correct
	if int(hdr.Len) != len(data) {
		return Header{}, nil, fmt.Errorf("data length is not the same as that written in header")
	}

	// checksum not supported to other host
	if hdr.Checksum == 0 {
		return hdr, data[HeaderSize:], nil
	}

	// caluculate checksum
	pseudoHdr := PseudoHeader{
		Src:  src,
		Dst:  dst,
		Type: ip.ProtoUDP,
		Len:  hdr.Len,
	}
	var w bytes.Buffer
	binary.Write(&w, binary.BigEndian, pseudoHdr)
	chksum := utils.CheckSum(w.Bytes(), 0)
	chksum = utils.CheckSum(data, uint32(^chksum))
	if chksum != 0 && chksum != 0xffff {
		return Header{}, nil, fmt.Errorf("checksum error (UDP)")
	}

	return hdr, data[HeaderSize:], nil
}

func header2data(hdr *Header, payload []byte, src ip.Addr, dst ip.Addr) ([]byte, error) {

	// pseudo header for caluculating checksum afterwards
	pseudoHdr := PseudoHeader{
		Src:  src,
		Dst:  dst,
		Type: ip.ProtoUDP,
		Len:  uint16(HeaderSize + len(payload)),
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
	return buf[PseudoHeaderSize:], nil
}

/*
	UDP Protocol
*/
// Protocol is struct for UDP protocol handler.
// This implements IPUpperProtocol interface.
type Protocol struct{}

func (p *Protocol) Type() ip.ProtoType {
	return ip.ProtoUDP
}

func (p *Protocol) RxHandler(data []byte, src ip.Addr, dst ip.Addr, ipIface *ip.Iface) error {
	hdr, payload, err := data2header(data, src, dst)
	if err != nil {
		return err
	}
	log.Printf("[D] UDP rxHandler: src=%s:%d,dst=%s:%d,iface=%s,udp header=%s,payload=%v", src, hdr.Src, dst, hdr.Dst, ipIface.Family(), hdr, payload)

	// search udp pcb whose address is dst
	mutex.Lock()
	defer mutex.Unlock()
	pcb := pcbSelect(dst, hdr.Dst)
	if pcb == nil {
		return fmt.Errorf("destination UDP protocol control block not found")
	}

	pcb.rxQueue <- buffer{
		foreign: Endpoint{
			Addr: src,
			Port: hdr.Src,
		},
		data: payload,
	}
	return nil
}

// TxHandler transmits UDP datagram to the other host.
func TxHandler(src Endpoint, dst Endpoint, data []byte) error {

	if len(data)+HeaderSize > ip.PayloadSizeMax {
		return fmt.Errorf("data size is too large for UDP payload")
	}

	// transform UDP header to byte strings
	hdr := Header{
		Src: src.Port,
		Dst: dst.Port,
		Len: uint16(HeaderSize + len(data)),
	}
	data, err := header2data(&hdr, data, src.Addr, dst.Addr)
	if err != nil {
		return err
	}

	log.Printf("[D] UDP TxHandler: src=%s,dst=%s,udp header=%s", src, dst, hdr)
	return ip.TxHandlerIP(ip.ProtoUDP, data, src.Addr, dst.Addr)
}

/*
	Protocol Control Block (for socket API)
*/

const (
	pcbStateFree    = 0
	pcbStateOpen    = 1
	pcbStateClosing = 2

	pcbBufSize = 100
)

var (
	mutex sync.Mutex
	pcbs  []*pcb
)

// pcb is protocol control block for UDP
type pcb struct {

	// pcb state
	state int

	// our UDP endpoint
	local Endpoint

	// receive queue
	rxQueue chan buffer
}

// buffer is
type buffer struct {

	// UDP endpoint of the source
	foreign Endpoint

	// data sent to us
	data []byte
}

func pcbSelect(address ip.Addr, port uint16) *pcb {
	for _, p := range pcbs {
		if p.local.Addr == address && p.local.Port == port {
			return p
		}
	}
	return nil
}

func OpenUDP() *pcb {
	pcb := &pcb{
		state: pcbStateOpen,
		local: Endpoint{
			Addr: ip.AddrAny,
		},
		rxQueue: make(chan buffer, pcbBufSize),
	}
	mutex.Lock()
	pcbs = append(pcbs, pcb)
	mutex.Unlock()
	return pcb
}

func Close(pcb *pcb) error {

	index := -1
	mutex.Lock()
	defer mutex.Unlock()
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

func (pcb *pcb) Bind(local Endpoint) error {

	// check if the same address has not been bound
	mutex.Lock()
	defer mutex.Unlock()
	for _, p := range pcbs {
		if p.local == local {
			return fmt.Errorf("local address(%s) is already binded", local)
		}
	}
	pcb.local = local
	log.Printf("[I] bound address local=%s", local)
	return nil
}

func (pcb *pcb) Send(data []byte, dst Endpoint) error {

	local := pcb.local

	if local.Addr == ip.AddrAny {
		route, err := ip.LookupTable(dst.Addr)
		if err != nil {
			return err
		}
		local.Addr = route.IpIface.Unicast
	}

	if local.Port == 0 { // zero value of Port (uint16)
		for p := PortMin; p <= PortMax; p++ {
			if pcbSelect(local.Addr, p) != nil {
				local.Port = p
				log.Printf("[D] registered UDP :address=%s,port=%d", local.Addr, local.Port)
				break
			}
		}
		if local.Port == 0 {
			return fmt.Errorf("there is no port number to assign")
		}
	}

	return TxHandler(local, dst, data)
}

// Listen listens data and write data to 'data'. if 'block' is false, there is no blocking I/O.
// This function returns data size,data,and source UDP endpoint.
func (pcb *pcb) Listen(block bool) (int, []byte, Endpoint) {

	if block {
		buf := <-pcb.rxQueue
		return len(buf.data), buf.data, buf.foreign
	}

	// no blocking
	select {
	case buf := <-pcb.rxQueue:
		return len(buf.data), buf.data, buf.foreign
	default:
		return 0, []byte{}, Endpoint{}
	}
}
