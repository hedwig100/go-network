package icmp

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"

	"github.com/hedwig100/go-network/ip"
	"github.com/hedwig100/go-network/utils"
)

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

/*
	ICMP Header
*/

const (
	ICMPHeaderSize int = 8
)

// ICMPHeader is a header for ICMP protocol
type ICMPHeader struct {

	// ICMP message type
	typ ICMPMessageType

	// code
	code ICMPMessageCode

	// checksum
	checksum uint16

	// message specific field
	values uint32
}

func (h ICMPHeader) String() string {
	switch h.typ {
	case ICMPTypeEchoReply, ICMPTypeEcho:
		return fmt.Sprintf(`
			typ: %s, 
			code: %d,
			checksum: %d,
			id: %d,
			seq: %d,
		`, h.typ, h.code, h.checksum, h.values>>16, h.values&0xff)
	default:
		return fmt.Sprintf(`
			typ: %s,
			code: %d,
			checksum: %d,
			values: %x,
		`, h.typ, h.code, h.checksum, h.values)
	}
}

func data2Header(b []byte) (ICMPHeader, []byte, error) {
	var hdr ICMPHeader
	r := bytes.NewReader(b)
	err := binary.Read(r, binary.BigEndian, &hdr)
	return hdr, b[ICMPHeaderSize:], err
}

func header2data(hdr ICMPHeader, data []byte) ([]byte, error) {

	w := bytes.NewBuffer(make([]byte, ICMPHeaderSize))
	err := binary.Write(w, binary.BigEndian, hdr)
	if err != nil {
		return nil, err
	}

	buf := make([]byte, ICMPHeaderSize+len(data))
	copy(buf[:ICMPHeaderSize], w.Bytes())
	copy(buf[ICMPHeaderSize:], data)

	// calculate checksum
	chksum := utils.CheckSum(buf[:ICMPHeaderSize])
	buf[2] = uint8(chksum >> 8)
	buf[3] = uint8(chksum)

	return buf, nil
}

/*
	ICMP Protocol
*/
type ICMPProtocol struct {
}

func (p *ICMPProtocol) Type() ip.IPProtocolType {
	return ip.IPProtocolICMP
}

func TxHandler(typ ICMPMessageType, code ICMPMessageCode, values uint32, data []byte, src ip.IPAddr, dst ip.IPAddr) error {

	hdr := ICMPHeader{
		typ:    typ,
		code:   code,
		values: values,
	}

	data, err := header2data(hdr, data)
	if err != nil {
		return err
	}

	log.Printf("[D] ICMP output: %s => %s,header=%s", src, dst, hdr) // TODO hdr.checksum should be set

	return ip.TxHandler(ip.IPProtocolICMP, data, src, dst)
}

func (p *ICMPProtocol) RxHandler(data []byte, src ip.IPAddr, dst ip.IPAddr, ipIface *ip.IPIface) error {

	if len(data) < ICMPHeaderSize {
		return fmt.Errorf("data size is too small for ICMP header")
	}

	chksum := utils.CheckSum(data[:ICMPHeaderSize])
	if chksum != 0 {
		return fmt.Errorf("checksum error in ICMP header")
	}

	hdr, data, err := data2Header(data)
	if err != nil {
		return err
	}

	log.Printf("[D] ICMP received: iface=%d,header=%s", ipIface.Family(), hdr)

	switch hdr.typ {
	case ICMPTypeEcho:
		if dst != ipIface.Unicast {
			// message addressed to broadcast address. responds with the address of the received interface
			dst = ipIface.Unicast
		}
		return TxHandler(ICMPTypeEchoReply, 0, hdr.values, data, dst, src)
	default:
		return fmt.Errorf("ICMP header type is unknown")
	}
}
