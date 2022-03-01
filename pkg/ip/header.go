package ip

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/hedwig100/go-network/pkg/utils"
)

/*
	IP Header
*/

// Header is header for IP packet.
type Header struct {

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
	ProtoType

	// checksum
	Checksum uint16

	// source IP address and destination IP address
	Src IPAddr
	Dst IPAddr
}

func (h Header) String() string {
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
	`, h.Vhl>>4, h.Vhl&0xf<<2, h.Tol, h.Tos, h.Id, h.Flags>>13, h.Flags&0x1fff, h.Ttl, h.ProtoType, h.Checksum, h.Src, h.Dst)
}

// data2IPHeader transforms byte strings to IP Header and the rest of the data
func data2headerIP(data []byte) (Header, []byte, error) {

	if len(data) < IPHeaderSizeMin {
		return Header{}, nil, fmt.Errorf("data size is too small")
	}

	// read header in bigEndian
	var ipHdr Header
	r := bytes.NewReader(data)
	err := binary.Read(r, binary.BigEndian, &ipHdr)
	if err != nil {
		return Header{}, nil, err
	}

	// check if the packet is iPv4
	if (ipHdr.Vhl >> 4) != IPVersionIPv4 {
		return Header{}, nil, err
	}

	// check header length and total length
	hlen := (ipHdr.Vhl & 0xf) << 2
	if uint8(len(data)) < hlen {
		return Header{}, nil, fmt.Errorf("data length is smaller than IHL")
	}
	if uint16(len(data)) < ipHdr.Tol {
		return Header{}, nil, fmt.Errorf("data length is smaller than Total Length")
	}

	// calculate checksum
	chksum := utils.CheckSum(data[:hlen], 0)
	if chksum != 0 && chksum != 0xffff { // 0 or -0
		return Header{}, nil, fmt.Errorf("checksum error (IP)")
	}

	return ipHdr, data[hlen:], nil
}

func header2dataIP(hdr *Header, payload []byte) ([]byte, error) {

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
	chksum := utils.CheckSum(buf[:IPHeaderSizeMin], 0)
	copy(buf[10:12], utils.Hton16(chksum))

	// set checksum in the header (for debug)
	hdr.Checksum = chksum
	return buf, nil
}
