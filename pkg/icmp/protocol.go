package icmp

import (
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

	return ip.TxHandler(ip.ProtoICMP, data, src, dst)
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
