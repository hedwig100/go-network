package device

import (
	"log"
	"math"
	"time"

	"github.com/hedwig100/go-network/pkg/net"
)

// Loopback is loopback device.
type Loopback struct {
	name  string
	flags uint16
}

// LoopbackInit reveices device name and returns loopback device.
func LoopbackInit(name string) *Loopback {
	l := &Loopback{
		name:  name,
		flags: net.DeviceFlagUp | net.DeviceFlagLoopback,
	}
	net.DeviceRegister(l)
	return l
}

func (l *Loopback) Name() string {
	return l.name
}

func (l *Loopback) Type() net.DeviceType {
	return net.DeviceTypeLoopback
}

func (l *Loopback) MTU() uint16 {
	return math.MaxUint16
}

func (l *Loopback) Flags() uint16 {
	return l.flags
}

func (l *Loopback) Addr() net.HardwareAddr {
	return nil
}

func (l *Loopback) AddIface(iface net.Interface) {
}

func (l *Loopback) Interfaces() []net.Interface {
	return []net.Interface{}
}

func (l *Loopback) Close() error {
	return nil
}

func (l *Loopback) Transmit(data []byte, typ net.ProtoType, dst net.HardwareAddr) error {

	// send back
	net.DeviceInputHanlder(typ, data, l)

	log.Printf("[I] Loopback TxHandler: data(%v) is trasmitted by loopback-device(name=%s)", data, l.name)
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
