package icmp

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"

	"github.com/hedwig100/go-network/pkg/ip"
	"github.com/hedwig100/go-network/pkg/utils"
)

// Init prepare the ICMP protocol
func Init() error {
	err := ip.ProtoRegister(&Proto{})
	return err
}

/*
	ICMP message type
*/

const (
	TypeEchoReply      MessageType = 0
	TypeDestUnreach    MessageType = 3
	TypeSourceQuench   MessageType = 4
	TypeRedirect       MessageType = 5
	TypeEcho           MessageType = 8
	TypeTimeExceeded   MessageType = 11
	TypeParamProblem   MessageType = 12
	TypeTimestamp      MessageType = 13
	TypeTimestampReply MessageType = 14
	TypeInfoRequest    MessageType = 15
	TypeInfoReply      MessageType = 16
)

type MessageType uint8

func (t MessageType) String() string {
	switch t {
	case 0:
		return "TypeEchoReply"
	case 3:
		return "TypeDestUnreach"
	case 4:
		return "TypeSourceQuench"
	case 5:
		return "TypeRedirect"
	case 8:
		return "TypeEcho"
	case 11:
		return "TypeTimeExceeded"
	case 12:
		return "TypeParamProblem"
	case 13:
		return "TypeTimestamp"
	case 14:
		return "TypeTimestampReply"
	case 15:
		return "TypeInfoRequest"
	case 16:
		return "TypeInfoReply"
	default:
		return "UNKNOWN"
	}
}

/*
	ICMP code
*/

const (
	// for Unreach
	CodeNetUnreach        MessageCode = 0
	CodeHostUnreach       MessageCode = 1
	CodeProtoUnreach      MessageCode = 2
	CodePortUnreach       MessageCode = 3
	CodeFragmentNeeded    MessageCode = 4
	CodeSourceRouteFailed MessageCode = 5

	// for Redirect
	CodeRedirectNet     MessageCode = 0
	CodeRedirectHost    MessageCode = 1
	CodeRedirectTosNet  MessageCode = 2
	CodeRedirectTosHost MessageCode = 3

	// for TimeExceeded
	CodeExceededTTL      MessageCode = 0
	CodeExceededFragment MessageCode = 1
)

type MessageCode uint8

func code2string(t MessageType, c MessageCode) string {
	switch t {
	case TypeDestUnreach:
		switch c {
		case CodeNetUnreach:
			return fmt.Sprintf("CodeNetUnreach(%d)", c)
		case CodeHostUnreach:
			return fmt.Sprintf("CodeHostUnreach(%d)", c)
		case CodeProtoUnreach:
			return fmt.Sprintf("CodeProtoUnreach(%d)", c)
		case CodePortUnreach:
			return fmt.Sprintf("CodePortUnreach(%d)", c)
		case CodeFragmentNeeded:
			return fmt.Sprintf("CodeFragmentNeeded(%d)", c)
		case CodeSourceRouteFailed:
			return fmt.Sprintf("CodeSourceRouteFailed(%d)", c)
		default:
			return fmt.Sprintf("UNKNOWN(%d)", c)
		}
	case TypeRedirect:
		switch c {
		case CodeRedirectNet:
			return fmt.Sprintf("CodeRedirectNet(%d)", c)
		case CodeRedirectHost:
			return fmt.Sprintf("CodeRedirectHost(%d)", c)
		case CodeRedirectTosNet:
			return fmt.Sprintf("CodeRedirectTosNet(%d)", c)
		case CodeRedirectTosHost:
			return fmt.Sprintf("CodeRedirectTosHost(%d)", c)
		default:
			return fmt.Sprintf("UNKNOWN(%d)", c)
		}
	case TypeTimeExceeded:
		switch c {
		case CodeExceededTTL:
			return fmt.Sprintf("CodeExceededTTL(%d)", c)
		case CodeExceededFragment:
			return fmt.Sprintf("CodeExceededFragment(%d)", c)
		default:
			return fmt.Sprintf("UNKNOWN(%d)", c)
		}
	default:
		return fmt.Sprintf("UNKNOWN(%d)", c)
	}
}

/*
	ICMP Header
*/

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

/*
	ICMP Protocol
*/
type Proto struct{}

func (p *Proto) Type() ip.ProtoType {
	return ip.ProtoICMP
}

func TxHandler(typ MessageType, code MessageCode, values uint32, data []byte, src ip.Addr, dst ip.Addr) error {

	hdr := Header{
		Typ:    typ,
		Code:   code,
		Values: values,
	}

	data, err := header2data(&hdr, data)
	if err != nil {
		return err
	}

	log.Printf("[D] ICMP TxHanlder: %s => %s,header=%s", src, dst, hdr)

	return ip.TxHandlerIP(ip.ProtoICMP, data, src, dst)
}

func (p *Proto) RxHandler(data []byte, src ip.Addr, dst ip.Addr, ipIface *ip.Iface) error {

	if len(data) < HeaderSize {
		return fmt.Errorf("data size is too small for ICMP header")
	}

	chksum := utils.CheckSum(data, 0)
	if chksum != 0 && chksum != 0xffff { // 0 or -0
		return fmt.Errorf("checksum error in ICMP header")
	}

	hdr, payload, err := data2header(data)
	if err != nil {
		return err
	}

	log.Printf("[D] ICMP rxHandler: iface=%d,header=%s", ipIface.Family(), hdr)

	switch hdr.Typ {
	case TypeEcho:
		if dst != ipIface.Unicast {
			// message addressed to broadcast address. responds with the address of the received interface
			dst = ipIface.Unicast
		}
		return TxHandler(TypeEchoReply, 0, hdr.Values, payload, dst, src)
	default:
		return fmt.Errorf("ICMP header type is unknown")
	}
}
