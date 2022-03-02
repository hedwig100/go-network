package net

import (
	"fmt"
	"log"
)

/*
	Family
*/

const (
	IfaceFamilyIP   IfaceFamily = 1
	IfaceFamilyIPv6 IfaceFamily = 2
)

type IfaceFamily uint8

func (f IfaceFamily) String() string {
	switch f {
	case IfaceFamilyIP:
		return "IPv4"
	case IfaceFamilyIPv6:
		return "IPv6"
	default:
		return "UNKNOWN"
	}
}

/*
	Interface
*/

var interfaces []Interface

// Interface is a logical interface,
// it serves as an entry point for devices and manages their addresses, etc
type Interface interface {

	// device to which the interface is tied
	Dev() Device

	// setters for tied devices
	SetDev(Device)

	// number which represents the kind of the interface
	Family() IfaceFamily
}

// IfaceRegister register iface to deev
func IfaceRegister(dev Device, iface Interface) error {

	// device cannot have the same family interface
	for _, registeredIface := range dev.Interfaces() {
		if registeredIface.Family() == iface.Family() {
			return fmt.Errorf("the same family(%s) interface is already registered to the device(%s)", iface.Family(), dev.Name())
		}
	}

	// add interface to the device
	dev.AddIface(iface)
	iface.SetDev(dev)
	interfaces = append(interfaces, iface)
	log.Printf("[I] iface=%s is registerd dev=%s", iface.Family(), dev.Name())
	return nil
}

// GetIface searches the family type of interface tied to the device
func GetIface(dev Device, family IfaceFamily) (Interface, error) {
	for _, iface := range dev.Interfaces() {
		if iface.Family() == family {
			return iface, nil
		}
	}
	return nil, fmt.Errorf("interface(family=%s) not found in device(dev=%s)", family, dev.Name())
}
