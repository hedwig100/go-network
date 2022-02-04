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

type Device interface {

	// name
	Name() string

	// device type ex) ethernet,loopback
	Type() DeviceType

	// Maximum Transmission Unit
	MTU() uint16

	// flag which represents state of the device
	Flags() uint16

	// logical interface that the device has
	Interfaces() []Interface

	// Open() error
	Close() error

	// output data to destination
	Transmit([]byte, ProtocolType, HardwareAddress) error

	// input from the device
	RxHandler(chan error)
}

func isUp(d Device) bool {
	return d.Flags()&NetDeviceFlagUp > 0
}

// すべてのデバイス: all the devices
var Devices []Device
var DevicesChannel []chan error

// デバイスを登録する
// register device
func DeviceRegister(dev Device) (err error) {

	// エラー通知用チャンネル: channel for error
	ch := make(chan error)
	DevicesChannel = append(DevicesChannel, ch)
	Devices = append(Devices, dev)

	// 受信ハンドラを起動させる: activate the receive handler
	go dev.RxHandler(ch)

	log.Printf("registerd dev=%s", dev.Name())
	return
}

// デバイスから入力されたデータをプロトコルに渡す
// passes the data input from the device to the protocol
func DeviceInputHanlder(typ ProtocolType, data []byte, dev Device) {
	for _, proto := range Protocols {
		if proto.Type() == typ {
			proto.RxHandler(data, dev)
			break
		}
	}
}

// デバイスからデータを出力する
// output the data from the device
func DeviceOutput(dev Device, data []byte, typ ProtocolType, dst HardwareAddress) (err error) {

	// デバイスが開いているか確認: check if the device is opening
	if !isUp(dev) {
		return fmt.Errorf("already closed dev=%s", dev.Name())
	}

	// データの長さがMTUを超えないことを確認: check if data length is longer than MTU
	if dev.MTU() < uint16(len(data)) {
		return fmt.Errorf("data size is too large dev=%s,mtu=%v", dev.Name(), dev.MTU())
	}

	err = dev.Transmit(data, typ, dst)
	return
}

// すべてのデバイスを開く: open all the devices
// func OpenDevices() (err error) {
// 	for _, dev := range Devices {

// 		if isUp(dev) {
// 			return fmt.Errorf("already opened dev=%s", dev.Name())
// 		}

// 		err = dev.Open()
// 		if err != nil {
// 			return
// 		}
// 		log.Printf("open device dev=%s", dev.Name())
// 	}
// 	return
// }

// すべてのデバイスを閉じる
// close all the devices
func CloseDevices() (err error) {
	for i, dev := range Devices {

		if !isUp(dev) {
			return fmt.Errorf("already closed dev=%s", dev.Name())
		}

		// チャンネルを閉じて受信ハンドラを停止: close the channel and stop the receive handler
		close(DevicesChannel[i])
		err = dev.Close()
		if err != nil {
			return
		}
		log.Printf("close device dev=%s", dev.Name())
	}
	return
}
