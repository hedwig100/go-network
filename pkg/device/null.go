package device

import (
	"log"
	"math"
	"time"

	"github.com/hedwig100/go-network/pkg/net"
)

type Null struct {
	name  string
	flags uint16
}

func NullInit(name string) *Null {
	n := &Null{
		name:  name,
		flags: net.NetDeviceFlagUp,
	}
	net.DeviceRegister(n)
	return n
}

func (n *Null) Name() string {
	return n.name
}

func (n *Null) Type() net.DeviceType {
	return net.NetDeviceTypeNull
}

func (n *Null) MTU() uint16 {
	return math.MaxUint16
}

func (n *Null) Flags() uint16 {
	return n.flags
}

func (n *Null) Address() net.HardwareAddress {
	return nil
}

func (n *Null) AddIface(iface net.Interface) {
}

func (n *Null) Interfaces() []net.Interface {
	return []net.Interface{}
}

func (n *Null) Close() error {
	return nil
}

func (n *Null) Transmit(data []byte, typ net.ProtocolType, dst net.HardwareAddress) error {
	log.Printf("[I] Null TxHandler: data(%v) is trasmitted by null-device(name=%s)", data, n.name)
	return nil
}

func (n *Null) RxHandler(done chan struct{}) {
	for {
		select {
		case <-done:
			return
		default:
			time.Sleep(time.Second)
		}
	}
}