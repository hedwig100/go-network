package net

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// UDPInit prepare the UDP protocol.
func UDPInit() {

}

/*
	UDP Header
*/

const (
	UDPHeaderSize       = 8
	UDPPseudoHeaderSize = 20 // only supports IPv4 now
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
	Src IPAddr

	// destination IP address
	Dst IPAddr

	// padding, always 0
	Pad uint8

	// protocol type, always 17(UDP)
	Type IPProtocolType

	// UDP packet length
	Len uint16

	// reald UDP header
	UDPHeader
}

// data2headerUDP transforms data to UDP header
// src,dst is used for caluculating checksum
func data2headerUDP(data []byte, src IPAddr, dst IPAddr) (UDPHeader, []byte, error) {

	if len(data) < UDPHeaderSize {
		return UDPHeader{}, nil, fmt.Errorf("data size is too small for udp header.")
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

	// caluculate checksum
	pseudoHdr := UDPPseudoHeader{
		Src:       src,
		Dst:       dst,
		Type:      17,
		Len:       uint16(len(data)),
		UDPHeader: hdr,
	}
	var w bytes.Buffer
	binary.Write(&w, binary.BigEndian, pseudoHdr)
	chksum := CheckSum(w.Bytes())
	if chksum != 0 && chksum != 0xffff {
		return UDPHeader{}, nil, fmt.Errorf("checksum error")
	}

	return hdr, data[UDPHeaderSize:], nil
}

func header2dataUDP(hdr *UDPHeader, payload []byte, src IPAddr, dst IPAddr) ([]byte, error) {

	// pseudo header for caluculating checksum afterwards
	pseudoHdr := UDPPseudoHeader{
		Src:       src,
		Dst:       dst,
		Type:      17,
		Len:       uint16(UDPHeaderSize + len(payload)),
		UDPHeader: *hdr,
	}

	// write header in bigEndian
	var w bytes.Buffer
	err := binary.Write(&w, binary.BigEndian, pseudoHdr)
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
	chksum := CheckSum(buf[:UDPPseudoHeaderSize])
	copy(buf[18:20], Hton16(chksum))

	// set checksum in the header (for debug)
	hdr.Checksum = chksum
	return buf[UDPPseudoHeaderSize-UDPHeaderSize:], nil
}

/*
	UDP Protocol
*/
// UDPProtocol is struct for UDP protocol handler.
// This implements IPUpperProtocol interface.
type UDPProtocol struct{}

func (p *UDPProtocol) Type() IPProtocolType {
	return IPProtocolUDP
}

// func (p *UDPProtocol) RxHandler(data []byte, src IPAddr, dst IPAddr, ipIface *IPIface) error {

// }

// // TxHandlerUDP transmits UDP datagram to the other host.
// func TxHandlerUDP() {

// }
