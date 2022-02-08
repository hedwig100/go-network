package net

import (
	"log"
	"math"
	"time"
)

type Null struct {
	name  string
	flags uint16
}

func NullInit(name string) (n *Null, err error) {
	n = &Null{
		name:  name,
		flags: NetDeviceFlagUp,
	}
	err = DeviceRegister(n)
	return
}

func (n *Null) Name() string {
	return n.name
}

func (n *Null) Type() DeviceType {
	return NetDeviceTypeNull
}

func (n *Null) MTU() uint16 {
	return math.MaxUint16
}

func (n *Null) Flags() uint16 {
	return n.flags
}

func (n *Null) Address() HardwareAddress {
	return nil
}

func (n *Null) AddIface(iface Interface) {
}

func (n *Null) Interfaces() []Interface {
	return nil
}

func (n *Null) Close() error {
	return nil
}

func (n *Null) Transmit(data []byte, typ ProtocolType, dst HardwareAddress) error {
	log.Printf("[I] data(%v) is trasmitted by null-device(name=%s)", data, n.name)
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