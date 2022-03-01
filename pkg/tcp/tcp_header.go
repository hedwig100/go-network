package tcp

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/hedwig100/go-network/pkg/ip"
	"github.com/hedwig100/go-network/pkg/udp"
	"github.com/hedwig100/go-network/pkg/utils"
)

/*
	TCP endpoint
*/

type Endpoint = udp.Endpoint

// Str2Endpoint encodes str to Endpoint
// ex) str="8.8.8.8:80"
func Str2Endpoint(str string) (Endpoint, error) {
	return udp.Str2Endpoint(str)
}

/*
	TCP Header
*/

const (
	CWR ControlFlag = 0b10000000
	ECE ControlFlag = 0b01000000
	URG ControlFlag = 0b00100000
	ACK ControlFlag = 0b00010000
	PSH ControlFlag = 0b00001000
	RST ControlFlag = 0b00000100
	SYN ControlFlag = 0b00000010
	FIN ControlFlag = 0b00000001
)

type ControlFlag uint8

func isSet(a ControlFlag, b ControlFlag) bool {
	return a&b > 0
}

func up(a ControlFlag, b ControlFlag, str string) string {
	if a&b > 0 {
		return str
	}
	return ""
}

func (f ControlFlag) String() string {
	return fmt.Sprintf(
		"%s%s%s%s%s%s%s%s",
		up(f, CWR, "CWR "),
		up(f, ECE, "ECE "),
		up(f, URG, "URG "),
		up(f, ACK, "ACK "),
		up(f, PSH, "PSH "),
		up(f, RST, "RST "),
		up(f, SYN, "SYN "),
		up(f, FIN, "FIN "),
	)
}

const (
	HeaderSizeMin    = 20
	PseudoHeaderSize = 12
)

// Header is header for TCP protocol
type Header struct {

	// source port number
	Src uint16

	// destination port number
	Dst uint16

	// sequence number
	Seq uint32

	// acknowledgement number
	Ack uint32

	// Offset is assembly of data offset(4bit) and reserved bit(4bit)
	Offset uint8

	// control flag
	Flag ControlFlag

	// window size
	Window uint16

	// checksum
	Checksum uint16

	// urgent pointer
	Urgent uint16
}

func (h Header) String() string {
	return fmt.Sprintf(`
		Dst: %d, 
		Src: %d,
		Seq: %d, 
		Ack: %d,
		Offset: %d,
		Control Flag: %s,
		Window Size: %d,
		Checksum: %x,
		Urgent Pointer: %x,
	`, h.Dst, h.Src, h.Seq, h.Ack, h.Offset>>4, h.Flag, h.Window, h.Checksum, h.Urgent)
}

// PseudoHeader is used for caluculating checksum.
type PseudoHeader struct {

	// source IP address
	Src ip.Addr

	// destination IP address
	Dst ip.Addr

	// padding, always 0
	Pad uint8

	// TCP protocol type,always 6
	Type ip.ProtoType

	// length of tcp packet
	Len uint16
}

// data2header transforms data to TCP header.
// returned []byte contains Options
// src,dst is used for caluculating checksum.
func data2header(data []byte, src ip.Addr, dst ip.Addr) (Header, []byte, error) {

	if len(data) < HeaderSizeMin {
		return Header{}, nil, fmt.Errorf("data size is too small for TCP Header")
	}

	// read header in bigEndian
	var hdr Header
	r := bytes.NewReader(data)
	err := binary.Read(r, binary.BigEndian, &hdr)
	if err != nil {
		return Header{}, nil, err
	}

	// caluculate checksum
	pseudoHdr := PseudoHeader{
		Src:  src,
		Dst:  dst,
		Type: ip.ProtoTCP,
		Len:  uint16(len(data)),
	}
	var w bytes.Buffer
	err = binary.Write(&w, binary.BigEndian, pseudoHdr)
	if err != nil {
		return Header{}, nil, err
	}
	chksum := utils.CheckSum(w.Bytes(), 0)
	chksum = utils.CheckSum(data, uint32(^chksum))
	if chksum != 0 && chksum != 0xffff {
		return Header{}, nil, fmt.Errorf("checksum error (TCP)")
	}

	return hdr, data[HeaderSizeMin:], nil
}

func header2data(hdr *Header, payload []byte, src ip.Addr, dst ip.Addr) ([]byte, error) {

	// pseudo header for caluculating checksum afterwards
	pseudoHdr := PseudoHeader{
		Src:  src,
		Dst:  dst,
		Type: ip.ProtoTCP,
		Len:  uint16(HeaderSizeMin + len(payload)),
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
	copy(buf[28:30], utils.Hton16(chksum)) // considering TCPPseudoHeaderSize

	// set checksum in the header (for debug)
	hdr.Checksum = chksum
	return buf[PseudoHeaderSize:], nil
}
