package ip

import "github.com/hedwig100/go-network/pkg/net"

/*
	IP logical Interface
*/

// Iface is IP interface
// *Iface implements net.Interface
type Iface struct {

	// device of the interface
	dev net.Device

	// unicast address ex) 192.0.0.1
	Unicast IPAddr

	// netmask ex) 255.255.255.0
	netmask IPAddr

	// broadcast address for the subnet
	broadcast IPAddr
}

func (i *Iface) Dev() net.Device {
	return i.dev
}

func (i *Iface) SetDev(dev net.Device) {
	i.dev = dev
}

func (i *Iface) Family() net.IfaceFamily {
	return net.IfaceFamilyIP
}

// NewIface returns Iface whose address is unicastStr
func NewIface(unicastStr string, netmaskStr string) (iface *Iface, err error) {

	unicast, err := Str2IPAddr(unicastStr)
	if err != nil {
		return
	}

	netmask, err := Str2IPAddr(netmaskStr)
	if err != nil {
		return
	}

	iface = &Iface{
		Unicast:   IPAddr(unicast),
		netmask:   IPAddr(netmask),
		broadcast: IPAddr(unicast | ^netmask),
	}
	return
}

// IfaceRegister registers ipIface to dev
func IfaceRegister(dev net.Device, ipIface *Iface) {
	net.IfaceRegister(dev, ipIface)

	// register subnet's routing information to routing table
	// this information is used when data is sent to the subnet's host
	RouteAdd(ipIface.Unicast&ipIface.netmask, ipIface.netmask, IPAddrAny, ipIface)
}
