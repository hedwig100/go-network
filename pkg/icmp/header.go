package icmp

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/hedwig100/go-network/pkg/utils"
)

const (
	HeaderSize int = 8
)

// Header is a header for ICMP protocol
type Header struct {

	// ICMP message type
	Typ MessageType

	// code
	Code MessageCode

	// checksum
	Checksum uint16

	// message specific field
	Values uint32
}

func (h Header) String() string {
	switch h.Typ {
	case TypeEchoReply, TypeEcho:
		return fmt.Sprintf(`
		typ: %s, 
		code: %s,
		checksum: %x,
		id: %d,
		seq: %d,
	`, h.Typ, code2string(h.Typ, h.Code), h.Checksum, h.Values>>16, h.Values&0xff)
	default:
		return fmt.Sprintf(`
		typ: %s,
		code: %s,
		checksum: %x,
		values: %x,
	`, h.Typ, code2string(h.Typ, h.Code), h.Checksum, h.Values)
	}
}

func data2header(data []byte) (Header, []byte, error) {

	// read header in bigEndian
	var hdr Header
	r := bytes.NewReader(data)
	err := binary.Read(r, binary.BigEndian, &hdr)

	// return header and payload and error
	return hdr, data[HeaderSize:], err
}

func header2data(hdr *Header, payload []byte) ([]byte, error) {

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

	// calculate checksum
	buf := w.Bytes()
	chksum := utils.CheckSum(buf[:HeaderSize], 0)
	copy(buf[2:4], utils.Hton16(chksum))

	// set checksum in the header (for debug)
	hdr.Checksum = chksum
	return buf, nil
}
