package net

import (
	"fmt"
	"log"
)

type DeviceType uint16
type HardwareAddress []uint8

const (
	NetDeviceTypeNull     DeviceType = 0x0000
	NetDeviceTypeLoopback DeviceType = 0x0001
	NetDeviceTypeEthernet DeviceType = 0x0002

	NetDeviceFlagUp        uint16 = 0x0001
	NetDeviceFlagLoopback  uint16 = 0x0010
	NetDeviceFlagBroadcast uint16 = 0x0020
	NetDeviceFlagP2P       uint16 = 0x0040
	NetDeviceFlagNeedARP   uint16 = 0x0100

	NetDeviceAddrLen = 16
)

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

	// add logical interface
	AddIface(Interface)

	// logical interface that the device has
	Interfaces() []Interface

	// Open() error
	Close() error

	// output data to destination
	Transmit([]byte, ProtocolType, HardwareAddress) error

	// input from the device
	RxHandler(chan struct{})
}

func isUp(d Device) bool {
	return d.Flags()&NetDeviceFlagUp > 0
}

var Devices []Device

// DeviceRegister registers the device
func DeviceRegister(dev Device) (err error) {

	// add the device
	Devices = append(Devices, dev)

	// activate the receive handler
	go dev.RxHandler(done)
	log.Printf("registerd dev=%s", dev.Name())
	return
}

// DeviceInputHandler receives data from the device and passes it to the protocol.
func DeviceInputHanlder(typ ProtocolType, data []byte, dev Device) {
	log.Printf("input data dev=%s,typ=%s,data:%v", dev, typ, data)

	for i, proto := range Protocols {
		if proto.Type() == typ {
			ProtocolBuffers[i] <- ProtocolBuffer{
				Data: data,
				Dev:  dev,
			}
			break
		}
	}
}

// DeviceOutput outputs the data from the device
func DeviceOutput(dev Device, data []byte, typ ProtocolType, dst HardwareAddress) (err error) {

	// check if the device is opening
	if !isUp(dev) {
		return fmt.Errorf("already closed dev=%s", dev.Name())
	}

	// check if data length is longer than MTU
	if dev.MTU() < uint16(len(data)) {
		return fmt.Errorf("data size is too large dev=%s,mtu=%v", dev.Name(), dev.MTU())
	}

	err = dev.Transmit(data, typ, dst)
	return
}

// CloseDevices closes all the devices
func CloseDevices() (err error) {

	// stop RxHandler beforehand
	close(done)

	for _, dev := range Devices {

		if !isUp(dev) {
			return fmt.Errorf("already closed dev=%s", dev.Name())
		}

		// close the channel and stop the receive handler
		err = dev.Close()
		if err != nil {
			return
		}
		log.Printf("close device dev=%s", dev.Name())
	}
	return
}
