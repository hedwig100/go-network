package devices

import (
	"bytes"
	"encoding/binary"
	"io"
	"log"
	"time"

	"github.com/hedwig100/go-network/net"
	"github.com/hedwig100/go-network/utils"
)

const (
	EtherAddrLen        = 6
	EtherAddrStrLen     = 18 /* "xx:xx:xx:xx:xx:xx\0" */
	EtherHdrSize        = 14
	EtherFrameSizeMin   = 60   /* without FCS */
	EtherFrameSizeMax   = 1514 /* without FCS */
	EtherPayloadSizeMin = (EtherFrameSizeMin - EtherHdrSize)
	EtherPayloadSizeMax = (EtherFrameSizeMax - EtherHdrSize)
)

var (
	EtherAddrAny       = [EtherAddrLen]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	EtherAddrBroadcast = [EtherAddrLen]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
)

type EthernetDevice struct {
	name       string
	flags      uint16
	interfaces []net.Interface
	Addr       [EtherAddrLen]byte
	fd         io.ReadWriteCloser
}

type EthernetHdr struct {
	Src  [EtherAddrLen]byte
	Dst  [EtherAddrLen]byte
	Type net.ProtocolType
}

func EtherInit(name string) (e EthernetDevice, err error) {

	// tapを開く: open tap
	name, fd, err := openTap(name)
	if err != nil {
		return
	}

	// ハードウェアアドレスを取得: get the hardware address
	addr, err := getAddr(name)
	if err != nil {
		return
	}

	e = EthernetDevice{
		name:  name,
		flags: net.NetDeviceFlagBroadcast | net.NetDeviceFlagNeedARP,
		Addr:  addr,
		fd:    fd,
	}
	return
}

func (e EthernetDevice) Name() string {
	return e.name
}

func (e EthernetDevice) Type() net.DeviceType {
	return net.NetDeviceTypeEthernet
}

func (e EthernetDevice) MTU() uint16 {
	return EtherPayloadSizeMax
}

func (e EthernetDevice) Flags() uint16 {
	return e.flags
}

func (e EthernetDevice) Interfaces() []net.Interface {
	return e.interfaces
}

// func (e EthernetDevice) Open() error {
// 	name, fd, err := OpenTap(e.name)
// 	if err != nil {
// 		return err
// 	}
// 	e.fd = fd
// 	e.name = name
// 	return nil
// }

func (e EthernetDevice) Close() error {
	err := e.fd.Close()
	return err
}

// データをデバイスを用いて送信する
// transmit the data using the device
func (e EthernetDevice) Transmit(data []byte, typ net.ProtocolType, dst net.HardwareAddress) error {

	// バッファーにEthernetヘッダの状態,データを入れる: put the status of the Ethernet header and data into the buffer
	var buf = make([]byte, EtherFrameSizeMax)
	copy(buf[:EtherAddrLen], dst)
	copy(buf[EtherAddrLen:2*EtherAddrLen], e.Addr[:EtherAddrLen])
	copy(buf[2*EtherAddrLen:EtherHdrSize], utils.Hton16(uint16(typ))) // littleEndian -> bigEndian
	copy(buf[EtherHdrSize:], data)

	_, err := e.fd.Write(buf)
	if err != nil {
		return err
	}

	log.Printf("data(%v) is trasmitted by ethernet-device(name=%s)", data, e.name)
	return nil
}

// ポーリングで入力があるか確認する, あればプロトコルへ渡す
// check if there is input by polling, if so, pass it to the protocol
func (e EthernetDevice) RxHandler(ch chan error) {
	buf := make([]byte, EtherFrameSizeMax)
	var rdr *bytes.Reader
	var hdr EthernetHdr

	for _, ok := <-ch; ok; {

		// デバイスファイルから読み出す: read from device file
		len, err := e.fd.Read(buf)
		if err != nil {
			ch <- err
		}

		if len == 0 {

			// データがなかったらビジーウェイトを防ぐために少しスリープ: if there is no data, it sleeps a little to prevent busy wait
			time.Sleep(time.Microsecond)

		} else if len < 14 { // EtherAddrLen + EtherAddrlen + 2(Type)

			// lenがイーサネットヘッダより小さい場合は無視: ignore the data if length is smaller than Ethernet Header Size
			log.Printf("[E] frame size is too small")

		} else {

			// データをビッグエンディアンで読み出す: read data with bigEndian
			rdr = bytes.NewReader(buf)
			err = binary.Read(rdr, binary.BigEndian, &hdr)
			if err != nil {
				ch <- err
			}

			// 自分宛のアドレスか確認: check if the address is for me
			if hdr.Dst != e.Addr && hdr.Dst != EtherAddrBroadcast {
				continue
			}

			// ヘッダ以降をデータとして上位プロトコルに渡す: pass the header and subsequent parts as data to the protocol
			log.Printf("[D] dev=%s,protocolType=%s,len=%d", e.name, hdr.Type, len)
			net.DeviceInputHanlder(hdr.Type, buf[14:len], e)
		}
	}
}
