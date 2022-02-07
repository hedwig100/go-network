package net

import (
	"fmt"
	"log"
)

const (
	NetIfaceFamilyIP   = 1
	NetIfaceFamilyIPv6 = 2
)

var Interfaces []Interface

/*
	Interface
*/
// Interfaces is a logical interface,
// it serves as an entry point for devices and manages their addresses, etc
type Interface interface {

	// device to which the interface is tied
	Dev() Device

	// setters for tied devices
	SetDev(Device)

	// number which represents the kind of the interface
	Family() int
}

// IfaceRegister register iface to deev
func IfaceRegister(dev Device, iface Interface) {
	dev.AddIface(iface)
	iface.SetDev(dev)
	Interfaces = append(Interfaces, iface)
	log.Printf("[I] iface=%d is registerd dev=%s", iface.Family(), dev.Name())
}

// GetIface searches the family type of interface tied to the device
func GetIface(dev Device, family int) (Interface, error) {
	for _, iface := range dev.Interfaces() {
		if iface.Family() == family {
			return iface, nil
		}
	}
	return nil, fmt.Errorf("interface(family=%d) not found in device(dev=%s)", family, dev.Name())
}
