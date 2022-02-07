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

func LoopbackInit(name string) (l *Loopback, err error) {
	l = &Loopback{
		name:  name,
		flags: net.NetDeviceFlagUp | net.NetDeviceFlagLoopback,
	}
	err = net.DeviceRegister(l)
	return
}

func (l *Loopback) Name() string {
	return l.name
}

func (l *Loopback) Type() net.DeviceType {
	return net.NetDeviceTypeLoopback
}

func (l *Loopback) MTU() uint16 {
	return math.MaxUint16
}

func (l *Loopback) Flags() uint16 {
	return l.flags
}

func (l *Loopback) AddIface(iface net.Interface) {
	log.Printf("[I] iface=%d is registerd dev=%s", iface.Family(), l.name)
}

func (l *Loopback) Interfaces() []net.Interface {
	return nil
}

func (l *Loopback) Close() error {
	return nil
}

func (l *Loopback) Transmit(data []byte, typ net.ProtocolType, dst net.HardwareAddress) error {

	// send back
	net.DeviceInputHanlder(typ, data, l)

	log.Printf("data(%v) is trasmitted by loopback-device(name=%s)", data, l.name)
	return nil
}

func (l *Loopback) RxHandler(done chan struct{}) {
	for {
		select {
		case <-done:
			return
		default:
			time.Sleep(time.Second)
		}
	}
}
