package udp

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/hedwig100/go-network/pkg/ip"
	"github.com/hedwig100/go-network/pkg/utils"
)

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
