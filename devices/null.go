package devices

import (
	"log"
	"math"
	"time"

	"github.com/hedwig100/go-network/net"
)

type Null struct {
	name  string
	flags uint16
}

func (n Null) Name() string {
	return n.name
}

func (n Null) Type() net.DeviceType {
	return net.NetDeviceTypeNull
}

func (n Null) MTU() uint16 {
	return math.MaxUint16
}

func (n Null) Flags() uint16 {
	return n.flags
}

func (n Null) Interfaces() []net.Interface {
	return nil
}

// func (n Null) Open() error {
// 	return nil
// }

func (n Null) Close() error {
	return nil
}

func (n Null) Transmit(data []byte, typ net.Protocol, dst net.HardwareAddress) error {
	log.Printf("data(%v) is trasmitted by null-device(name=%s)", data, n.name)
	return nil
}

func (n Null) RxHandler(ch chan error) {
	for _, ok := <-ch; ok; {
		time.Sleep(time.Second)
	}
}
