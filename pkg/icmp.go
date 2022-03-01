package pkg

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"

	"github.com/hedwig100/go-network/pkg/ip"
)

// icmpInit prepare the ICMP protocol
func icmpInit() error {
	err := ip.IPProtocolRegister(&ICMPProtocol{})
	return err
}

/*
	ICMP message type
*/

const (
	ICMPTypeEchoReply      ICMPMessageType = 0
	ICMPTypeDestUnreach    ICMPMessageType = 3
	ICMPTypeSourceQuench   ICMPMessageType = 4
	ICMPTypeRedirect       ICMPMessageType = 5
	ICMPTypeEcho           ICMPMessageType = 8
	ICMPTypeTimeExceeded   ICMPMessageType = 11
	ICMPTypeParamProblem   ICMPMessageType = 12
	ICMPTypeTimestamp      ICMPMessageType = 13
	ICMPTypeTimestampReply ICMPMessageType = 14
	ICMPTypeInfoRequest    ICMPMessageType = 15
	ICMPTypeInfoReply      ICMPMessageType = 16
)

type ICMPMessageType uint8

func (t ICMPMessageType) String() string {
	switch t {
	case 0:
		return "ICMPTypeEchoReply"
	case 3:
		return "ICMPTypeDestUnreach"
	case 4:
		return "ICMPTypeSourceQuench"
	case 5:
		return "ICMPTypeRedirect"
	case 8:
		return "ICMPTypeEcho"
	case 11:
		return "ICMPTypeTimeExceeded"
	case 12:
		return "ICMPTypeParamProblem"
	case 13:
		return "ICMPTypeTimestamp"
	case 14:
		return "ICMPTypeTimestampReply"
	case 15:
		return "ICMPTypeInfoRequest"
	case 16:
		return "ICMPTypeInfoReply"
	default:
		return "UNKNOWN"
	}
}

/*
	ICMP code
*/

const (
	// for Unreach
	ICMPCodeNetUnreach        ICMPMessageCode = 0
	ICMPCodeHostUnreach       ICMPMessageCode = 1
	ICMPCodeProtoUnreach      ICMPMessageCode = 2
	ICMPCodePortUnreach       ICMPMessageCode = 3
	ICMPCodeFragmentNeeded    ICMPMessageCode = 4
	ICMPCodeSourceRouteFailed ICMPMessageCode = 5

	// for Redirect
	ICMPCodeRedirectNet     ICMPMessageCode = 0
	ICMPCodeRedirectHost    ICMPMessageCode = 1
	ICMPCodeRedirectTosNet  ICMPMessageCode = 2
	ICMPCodeRedirectTosHost ICMPMessageCode = 3

	// for TimeExceeded
	ICMPCodeExceededTTL      ICMPMessageCode = 0
	ICMPCodeExceededFragment ICMPMessageCode = 1
)

type ICMPMessageCode uint8

func code2string(t ICMPMessageType, c ICMPMessageCode) string {
	switch t {
	case ICMPTypeDestUnreach:
		switch c {
		case ICMPCodeNetUnreach:
			return fmt.Sprintf("ICMPCodeNetUnreach(%d)", c)
		case ICMPCodeHostUnreach:
			return fmt.Sprintf("ICMPCodeHostUnreach(%d)", c)
		case ICMPCodeProtoUnreach:
			return fmt.Sprintf("ICMPCodeProtoUnreach(%d)", c)
		case ICMPCodePortUnreach:
			return fmt.Sprintf("ICMPCodePortUnreach(%d)", c)
		case ICMPCodeFragmentNeeded:
			return fmt.Sprintf("ICMPCodeFragmentNeeded(%d)", c)
		case ICMPCodeSourceRouteFailed:
			return fmt.Sprintf("ICMPCodeSourceRouteFailed(%d)", c)
		default:
			return fmt.Sprintf("UNKNOWN(%d)", c)
		}
	case ICMPTypeRedirect:
		switch c {
		case ICMPCodeRedirectNet:
			return fmt.Sprintf("ICMPCodeRedirectNet(%d)", c)
		case ICMPCodeRedirectHost:
			return fmt.Sprintf("ICMPCodeRedirectHost(%d)", c)
		case ICMPCodeRedirectTosNet:
			return fmt.Sprintf("ICMPCodeRedirectTosNet(%d)", c)
		case ICMPCodeRedirectTosHost:
			return fmt.Sprintf("ICMPCodeRedirectTosHost(%d)", c)
		default:
			return fmt.Sprintf("UNKNOWN(%d)", c)
		}
	case ICMPTypeTimeExceeded:
		switch c {
		case ICMPCodeExceededTTL:
			return fmt.Sprintf("ICMPCodeExceededTTL(%d)", c)
		case ICMPCodeExceededFragment:
			return fmt.Sprintf("ICMPCodeExceededFragment(%d)", c)
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
	ICMPHeaderSize int = 8
)

// ICMPHeader is a header for ICMP protocol
type ICMPHeader struct {

	// ICMP message type
	Typ ICMPMessageType

	// code
	Code ICMPMessageCode

	// checksum
	Checksum uint16

	// message specific field
	Values uint32
}

func (h ICMPHeader) String() string {
	switch h.Typ {
	case ICMPTypeEchoReply, ICMPTypeEcho:
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

func data2headerICMP(data []byte) (ICMPHeader, []byte, error) {

	// read header in bigEndian
	var hdr ICMPHeader
	r := bytes.NewReader(data)
	err := binary.Read(r, binary.BigEndian, &hdr)

	// return header and payload and error
	return hdr, data[ICMPHeaderSize:], err
}

func header2dataICMP(hdr *ICMPHeader, payload []byte) ([]byte, error) {

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
	chksum := CheckSum(buf[:ICMPHeaderSize], 0)
	copy(buf[2:4], Hton16(chksum))

	// set checksum in the header (for debug)
	hdr.Checksum = chksum
	return buf, nil
}

/*
	ICMP Protocol
*/
type ICMPProtocol struct{}

func (p *ICMPProtocol) Type() ip.IPProtocolType {
	return ip.IPProtocolICMP
}

func TxHandlerICMP(typ ICMPMessageType, code ICMPMessageCode, values uint32, data []byte, src ip.IPAddr, dst ip.IPAddr) error {

	hdr := ICMPHeader{
		Typ:    typ,
		Code:   code,
		Values: values,
	}

	data, err := header2dataICMP(&hdr, data)
	if err != nil {
		return err
	}

	log.Printf("[D] ICMP TxHanlder: %s => %s,header=%s", src, dst, hdr)

	return ip.TxHandlerIP(ip.IPProtocolICMP, data, src, dst)
}

func (p *ICMPProtocol) RxHandler(data []byte, src ip.IPAddr, dst ip.IPAddr, ipIface *ip.IPIface) error {

	if len(data) < ICMPHeaderSize {
		return fmt.Errorf("data size is too small for ICMP header")
	}

	chksum := CheckSum(data, 0)
	if chksum != 0 && chksum != 0xffff { // 0 or -0
		return fmt.Errorf("checksum error in ICMP header")
	}

	hdr, payload, err := data2headerICMP(data)
	if err != nil {
		return err
	}

	log.Printf("[D] ICMP rxHandler: iface=%d,header=%s", ipIface.Family(), hdr)

	switch hdr.Typ {
	case ICMPTypeEcho:
		if dst != ipIface.Unicast {
			// message addressed to broadcast address. responds with the address of the received interface
			dst = ipIface.Unicast
		}
		return TxHandlerICMP(ICMPTypeEchoReply, 0, hdr.Values, payload, dst, src)
	default:
		return fmt.Errorf("ICMP header type is unknown")
	}
}
