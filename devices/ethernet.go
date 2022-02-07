package devices

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/hedwig100/go-network/net"
	"github.com/hedwig100/go-network/utils"
)

const (
	EtherAddrLen    = 6
	EtherAddrStrLen = 17 /* "xx:xx:xx:xx:xx:xx" */
	EtherHdrSize    = 14

	EtherFrameSizeMin = 60   /* without FCS */
	EtherFrameSizeMax = 1514 /* without FCS */

	EtherPayloadSizeMin = (EtherFrameSizeMin - EtherHdrSize)
	EtherPayloadSizeMax = (EtherFrameSizeMax - EtherHdrSize)
)

var (
	EtherAddrAny       = EthernetAddress{addr: [EtherAddrLen]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00}}
	EtherAddrBroadcast = EthernetAddress{addr: [EtherAddrLen]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}}
)

// EtherInit setup ethernet device
func EtherInit(name string) (e *EthernetDevice, err error) {

	// open tap
	name, file, err := openTap(name)
	if err != nil {
		return
	}

	// get the hardware address
	addr, err := getAddr(name)
	if err != nil {
		return
	}

	e = &EthernetDevice{
		name:  name,
		flags: net.NetDeviceFlagBroadcast | net.NetDeviceFlagNeedARP | net.NetDeviceFlagUp,
		EthernetAddress: EthernetAddress{
			addr: addr,
		},
		file: file,
	}

	err = net.DeviceRegister(e)
	return
}

/*
	Ethernet Address (MAC address)
*/

// EthernetAddress implments net.HardwareAddress interface
type EthernetAddress struct {
	addr [EtherAddrLen]byte
}

func (a EthernetAddress) Address() []byte {
	return a.addr[:]
}

func (a EthernetAddress) String() string {
	return fmt.Sprintf("%x:%x:%x:%x:%x:%x", a.addr[0], a.addr[1], a.addr[2], a.addr[3], a.addr[4], a.addr[5])
}

/*
	Ethernet Header
*/

// EthernetHdr is header for ethernet frame
type EthernetHdr struct {

	// source address
	Src EthernetAddress

	// destination address
	Dst EthernetAddress

	// protocol type
	Type net.ProtocolType
}

/*
	Ethernet Device
*/

// EthernetDevice implements net.Device interface
type EthernetDevice struct {

	// name
	name string

	// flags represents device state and device type
	flags uint16

	// interfaces tied to the device
	interfaces []net.Interface

	// ethernet address
	EthernetAddress

	// device file (character file)
	file io.ReadWriteCloser
}

func (e *EthernetDevice) Name() string {
	return e.name
}

func (e *EthernetDevice) Type() net.DeviceType {
	return net.NetDeviceTypeEthernet
}

func (e *EthernetDevice) MTU() uint16 {
	return EtherPayloadSizeMax
}

func (e *EthernetDevice) Flags() uint16 {
	return e.flags
}

func (e *EthernetDevice) Address() net.HardwareAddress {
	return e.EthernetAddress
}

func (e *EthernetDevice) AddIface(iface net.Interface) {
	log.Printf("[I] iface=%d is registerd dev=%s", iface.Family(), e.name)
	e.interfaces = append(e.interfaces, iface)
}

func (e *EthernetDevice) Interfaces() []net.Interface {
	return e.interfaces
}

func (e *EthernetDevice) Close() error {
	err := e.file.Close()
	return err
}

func (e *EthernetDevice) Transmit(data []byte, typ net.ProtocolType, dst net.HardwareAddress) error {

	// dst must be Ethernet address
	etherDst, ok := dst.(EthernetAddress)
	if !ok {
		return fmt.Errorf("ethernet device only supports ethernet address")
	}

	// put the status of the Ethernet header and data into the buffer
	var buf = make([]byte, EtherFrameSizeMax)
	copy(buf[:EtherAddrLen], etherDst.addr[:])
	copy(buf[EtherAddrLen:2*EtherAddrLen], e.addr[:])
	copy(buf[2*EtherAddrLen:EtherHdrSize], utils.Hton16(uint16(typ))) // littleEndian -> bigEndian
	copy(buf[EtherHdrSize:], data)

	_, err := e.file.Write(buf)
	if err != nil {
		return err
	}

	log.Printf("data(%v) is trasmitted by ethernet-device(name=%s)", data, e.name)
	return nil
}

func (e *EthernetDevice) RxHandler(done chan struct{}) {
	buf := make([]byte, EtherFrameSizeMax)
	var rdr *bytes.Reader
	var hdr EthernetHdr

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
			log.Printf("[E] dev=%s,%s", e.name, err.Error())
		}

		if len == 0 {

			// if there is no data, it sleeps a little to prevent busy wait
			time.Sleep(time.Microsecond)

		} else if len < EtherHdrSize {

			// ignore the data if length is smaller than Ethernet Header Size
			log.Printf("[E] frame size is too small")

		} else {

			// read data with bigEndian
			rdr = bytes.NewReader(buf)
			err = binary.Read(rdr, binary.BigEndian, &hdr)
			if err != nil {
				log.Printf("[E] dev=%s,%s", e.name, err.Error())
			}

			// check if the address is for me
			if hdr.Dst.addr != e.addr && hdr.Dst != EtherAddrBroadcast {
				continue
			}

			// pass the header and subsequent parts as data to the protocol
			log.Printf("[D] dev=%s,protocolType=%s,len=%d", e.name, hdr.Type, len)
			net.DeviceInputHanlder(hdr.Type, buf[14:len], e)
		}

	}
}
