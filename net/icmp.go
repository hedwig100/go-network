package net

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
)

func ICMPInit() error {
	err := IPProtocolRegister(&ICMPProtocol{})
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
		code: %d,
		checksum: %d,
		id: %d,
		seq: %d,
	`, h.Typ, h.Code, h.Checksum, h.Values>>16, h.Values&0xff)
	default:
		return fmt.Sprintf(`
		typ: %s,
		code: %d,
		checksum: %d,
		values: %x,
	`, h.Typ, h.Code, h.Checksum, h.Values)
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
	chksum := CheckSum(buf[:ICMPHeaderSize])
	copy(buf[2:4], Hton16(chksum))

	// set checksum in the header (for debug)
	hdr.Checksum = chksum
	return buf, nil
}

/*
	ICMP Protocol
*/
type ICMPProtocol struct {
}

func (p *ICMPProtocol) Type() IPProtocolType {
	return IPProtocolICMP
}

func TxHandlerICMP(typ ICMPMessageType, code ICMPMessageCode, values uint32, data []byte, src IPAddr, dst IPAddr) error {

	hdr := ICMPHeader{
		Typ:    typ,
		Code:   code,
		Values: values,
	}

	data, err := header2dataICMP(&hdr, data)
	if err != nil {
		return err
	}

	log.Printf("[D] ICMP output: %s => %s,header=%s", src, dst, hdr) // TODO hdr.checksum should be set

	return TxHandlerIP(IPProtocolICMP, data, src, dst)
}

func (p *ICMPProtocol) RxHandler(data []byte, src IPAddr, dst IPAddr, ipIface *IPIface) error {

	if len(data) < ICMPHeaderSize {
		return fmt.Errorf("data size is too small for ICMP header")
	}

	chksum := CheckSum(data[:ICMPHeaderSize])
	if chksum != 0 {
		return fmt.Errorf("checksum error in ICMP header")
	}

	hdr, data, err := data2headerICMP(data)
	if err != nil {
		return err
	}

	log.Printf("[D] ICMP received: iface=%d,header=%s", ipIface.Family(), hdr)

	switch hdr.Typ {
	case ICMPTypeEcho:
		if dst != ipIface.Unicast {
			// message addressed to broadcast address. responds with the address of the received interface
			dst = ipIface.Unicast
		}
		return TxHandlerICMP(ICMPTypeEchoReply, 0, hdr.Values, data, dst, src)
	default:
		return fmt.Errorf("ICMP header type is unknown")
	}
}
