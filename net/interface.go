package net

import "fmt"

const (
	NetIfaceFamilyIP   = 1
	NetIfaceFamilyIPv6 = 2
)

var Interfaces []Interface

// 論理インタフェース, デバイスの入り口となりアドレスなどを管理する
// logical interface, serves as an entry point for devices and manages their addresses, etc
type Interface interface {

	// インタフェースが属するデバイス
	// device to which the interface is tied
	Dev() Device

	// 紐づくデバイスのセッター
	// setters for tied devices
	SetDev(Device)

	// インタフェースの種類を表す番号
	// number which represents the kind of the interface
	Family() int
}

// IfaceRegister register iface to deev
func IfaceRegister(dev Device, iface Interface) {
	dev.AddIface(iface)
	iface.SetDev(dev)
	Interfaces = append(Interfaces, iface)
}

// デバイスに紐づくあるインタフェースを探す,入力時に必要
// search certain interface tied to device, required for input
func GetIface(dev Device, family int) (Interface, error) {
	for _, iface := range dev.Interfaces() {
		if iface.Family() == family {
			return iface, nil
		}
	}
	return nil, fmt.Errorf("interface(family=%d) not found in device(dev=%s)", family, dev.Name())
}
