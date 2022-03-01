package device

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/hedwig100/go-network/pkg/device/raw"
	"github.com/hedwig100/go-network/pkg/net"
)

const (
	EtherAddrLen    = 6
	EtherAddrStrLen = 17 /* "xx:xx:xx:xx:xx:xx" */
	EtherHeaderSize = 14

	EtherFrameSizeMin = 60   /* without FCS */
	EtherFrameSizeMax = 1514 /* without FCS */

	EtherPayloadSizeMin = (EtherFrameSizeMin - EtherHeaderSize)
	EtherPayloadSizeMax = (EtherFrameSizeMax - EtherHeaderSize)
)

var (
	EtherAddrAny       = EtherAddr([EtherAddrLen]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
	EtherAddrBroadcast = EtherAddr([EtherAddrLen]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff})
)

// EtherInit setup ethernet device
func EtherInit(name string) (*Ether, error) {

	// open tap
	name, file, err := raw.OpenTap(name)
	if err != nil {
		return nil, err
	}

	// get the hardware address
	_addr, err := raw.GetAddr(name)
	if err != nil {
		return nil, err
	}

	// transform []byte to [EtherAddrLen]byte
	var addr [EtherAddrLen]byte
	copy(addr[:], _addr)

	e := &Ether{
		name:      name,
		flags:     net.NetDeviceFlagBroadcast | net.NetDeviceFlagNeedARP | net.NetDeviceFlagUp,
		EtherAddr: EtherAddr(addr),
		file:      file,
	}
	net.DeviceRegister(e)
	return e, nil
}

/*
	Ethernet Header
*/

// EtherHeader is header for ethernet frame
type EtherHeader struct {

	// source address
	Src EtherAddr

	// destination address
	Dst EtherAddr

	// protocol type
	Type net.ProtocolType
}

func (h EtherHeader) String() string {
	return fmt.Sprintf(`
		Src: %s,
		Dst: %s,
		Type: %s
	`, h.Src, h.Dst, h.Type)
}

func data2header(data []byte) (EtherHeader, []byte, error) {

	// read header in bigEndian
	var hdr EtherHeader
	r := bytes.NewReader(data[:EtherHeaderSize])
	err := binary.Read(r, binary.BigEndian, &hdr)

	// return header and payload and error
	return hdr, data[EtherHeaderSize:], err
}

func header2data(hdr EtherHeader, payload []byte) ([]byte, error) {

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

	return w.Bytes(), nil
}

/*
	Ethernet Device
*/

// Ether implements net.Device interface
type Ether struct {

	// name
	name string

	// flags represents device state and device type
	flags uint16

	// interfaces tied to the device
	interfaces []net.Interface

	// ethernet address
	EtherAddr

	// device file (character file)
	file io.ReadWriteCloser
}

func (e *Ether) Name() string {
	return e.name
}

func (e *Ether) Type() net.DeviceType {
	return net.NetDeviceTypeEthernet
}

func (e *Ether) MTU() uint16 {
	return EtherPayloadSizeMax
}

func (e *Ether) Flags() uint16 {
	return e.flags
}

func (e *Ether) Addr() net.HardwareAddr {
	return e.EtherAddr
}

func (e *Ether) AddIface(iface net.Interface) {
	e.interfaces = append(e.interfaces, iface)
}

func (e *Ether) Interfaces() []net.Interface {
	if e.interfaces == nil {
		return []net.Interface{}
	}
	return e.interfaces
}

func (e *Ether) Close() error {
	err := e.file.Close()
	return err
}

func (e *Ether) Transmit(data []byte, typ net.ProtocolType, dst net.HardwareAddr) error {

	// dst must be Ethernet address
	etherDst, ok := dst.(EtherAddr)
	log.Println(typ, dst)
	if !ok {
		return fmt.Errorf("ethernet device only supports ethernet address")
	}

	// put header and data into the data
	hdr := EtherHeader{
		Src:  e.EtherAddr,
		Dst:  etherDst,
		Type: typ,
	}
	data, err := header2data(hdr, data)
	if err != nil {
		return err
	}

	// write character file
	_, err = e.file.Write(data)
	if err != nil {
		return err
	}

	log.Printf("[D] Ether TxHandler: data is trasmitted by ethernet-device(name=%s),header=%s", e.name, hdr)
	return nil
}

func (e *Ether) RxHandler(done chan struct{}) {
	buf := make([]byte, EtherFrameSizeMax)

	for {

		// check if finished or not
		select {
		case <-done:
			return
		default:
		}

		// read from device file (character file)
		len, err := e.file.Read(buf)
		if err != nil {
			log.Printf("[E] Ether rxHandler: dev=%s,%s", e.name, err.Error())
			continue
		}

		if len == 0 {

			// if there is no data, it sleeps a little to prevent busy wait
			time.Sleep(time.Microsecond)

		} else if len < EtherHeaderSize {

			// ignore the data if length is smaller than Ethernet Header Size
			log.Printf("[E] Ether rxHandler: frame size is too small")
			time.Sleep(time.Microsecond)

		} else {

			// read data
			hdr, payload, err := data2header(buf)
			if err != nil {
				log.Printf("[E] dev=%s,%s", e.name, err.Error())
				continue
			}

			// check if the address is for me
			if hdr.Dst != e.EtherAddr && hdr.Dst != EtherAddrBroadcast {
				continue
			}

			// pass the header and subsequent parts as data to the protocol
			log.Printf("[D] Ether rxHandler: dev=%s,protocolType=%s,len=%d,header=%s", e.name, hdr.Type, len, hdr)
			net.DeviceInputHanlder(hdr.Type, payload[:len-EtherHeaderSize], e)
		}

	}
}
