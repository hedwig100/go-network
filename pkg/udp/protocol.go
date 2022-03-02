package udp

import (
	"fmt"
	"log"

	"github.com/hedwig100/go-network/pkg/ip"
)

// Init prepare the UDP protocol.
func Init() error {
	return ip.ProtoRegister(&Protocol{})
}

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
	return ip.TxHandler(ip.ProtoUDP, data, src.Addr, dst.Addr)
}
