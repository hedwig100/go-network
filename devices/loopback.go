package devices

import (
	"log"
	"math"
	"time"

	"github.com/hedwig100/go-network/net"
)

type Loopback struct {
	name  string
	flags uint16
}

func (l Loopback) Name() string {
	return l.name
}

func (l Loopback) Type() net.DeviceType {
	return net.NetDeviceTypeLoopback
}

func (l Loopback) MTU() uint16 {
	return math.MaxUint16
}

func (l Loopback) Flags() uint16 {
	return l.flags
}

func (l Loopback) Interfaces() []net.Interface {
	return nil
}

// func (l Loopback) Open() error {
// 	return nil
// }

func (l Loopback) Close() error {
	return nil
}

func (l Loopback) Transmit(data []byte, typ net.ProtocolType, dst net.HardwareAddress) error {

	// 送り返す send back
	net.DeviceInputHanlder(typ, data, l)

	log.Printf("data(%v) is trasmitted by loopback-device(name=%s)", data, l.name)
	return nil
}

func (l Loopback) RxHandler(ch chan error) {
	for _, ok := <-ch; ok; {
		time.Sleep(time.Second)
	}
}
