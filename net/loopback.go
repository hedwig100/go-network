package net

import (
	"log"
	"math"
	"time"
)

type Loopback struct {
	name  string
	flags uint16
}

func LoopbackInit(name string) (l *Loopback, err error) {
	l = &Loopback{
		name:  name,
		flags: NetDeviceFlagUp | NetDeviceFlagLoopback,
	}
	err = DeviceRegister(l)
	return
}

func (l *Loopback) Name() string {
	return l.name
}

func (l *Loopback) Type() DeviceType {
	return NetDeviceTypeLoopback
}

func (l *Loopback) MTU() uint16 {
	return math.MaxUint16
}

func (l *Loopback) Flags() uint16 {
	return l.flags
}

func (l *Loopback) Address() HardwareAddress {
	return nil
}

func (l *Loopback) AddIface(iface Interface) {
}

func (l *Loopback) Interfaces() []Interface {
	return nil
}

func (l *Loopback) Close() error {
	return nil
}

func (l *Loopback) Transmit(data []byte, typ ProtocolType, dst HardwareAddress) error {

	// send back
	DeviceInputHanlder(typ, data, l)

	log.Printf("[I] Tx data(%v) is trasmitted by loopback-device(name=%s)", data, l.name)
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
