package net

import (
	"fmt"
	"log"
)

type DeviceType uint16

const (
	DeviceTypeNull     DeviceType = 0x0000
	DeviceTypeLoopback DeviceType = 0x0001
	DeviceTypeEther    DeviceType = 0x0002

	DeviceFlagUp        uint16 = 0x0001
	DeviceFlagLoopback  uint16 = 0x0010
	DeviceFlagBroadcast uint16 = 0x0020
	DeviceFlagP2P       uint16 = 0x0040
	DeviceFlagNeedARP   uint16 = 0x0100
)

var devices []Device

/*
	Device
*/
// Device interface is the abstraction of the device
type Device interface {

	// name
	Name() string

	// device type ex) ethernet,loopback
	Type() DeviceType

	// Maximum Transmission Unit
	MTU() uint16

	// flag which represents state of the device
	Flags() uint16

	// device's hardware address
	Addr() HardwareAddr

	// add logical interface
	AddIface(Interface)

	// logical interface that the device has
	Interfaces() []Interface

	// Open() error
	Close() error

	// output data to destination
	TxHandler([]byte, ProtoType, HardwareAddr) error

	// input from the device
	RxHandler(chan struct{})
}

func isUp(d Device) bool {
	return d.Flags()&DeviceFlagUp > 0
}

// DeviceRegister registers the device
func DeviceRegister(dev Device) {
	devices = append(devices, dev)
	log.Printf("[I] registerd dev=%s", dev.Name())
}

// DeviceInputHandler receives data from the device and passes it to the protocol.
func DeviceInputHanlder(typ ProtoType, data []byte, dev Device) {
	log.Printf("[I] input data dev=%s,typ=%s,data:%v", dev, typ, data)

	for i, proto := range protos {
		if proto.Type() == typ {
			protoBuffers[i] <- ProtoBuffer{
				Data: data,
				Dev:  dev,
			}
			break
		}
	}
}

// DeviceOutput outputs the data from the device
func DeviceOutput(dev Device, data []byte, typ ProtoType, dst HardwareAddr) error {

	// check if the device is opening
	if !isUp(dev) {
		return fmt.Errorf("already closed dev=%s", dev.Name())
	}

	// check if data length is longer than MTU
	if dev.MTU() < uint16(len(data)) {
		return fmt.Errorf("data size is too large dev=%s,mtu=%v", dev.Name(), dev.MTU())
	}

	err := dev.TxHandler(data, typ, dst)
	if err != nil {
		return err
	}
	log.Printf("[I] device output, dev=%s,typ=%s", dev.Name(), typ)
	return nil
}
