package ip

import (
	"fmt"
	"log"

	"github.com/hedwig100/go-network/pkg/utils"
)

var routes []IPRoute

// IPRoute is routing table entry
type IPRoute struct {
	network IPAddr
	netmask IPAddr
	nexthop IPAddr
	IpIface *Iface
}

// RouteAdd add routing table entry to routing table
func RouteAdd(network IPAddr, netmask IPAddr, nexthop IPAddr, ipIface *Iface) {
	routes = append(routes, IPRoute{
		network: network,
		netmask: netmask,
		nexthop: nexthop,
		IpIface: ipIface,
	})
	log.Printf("[I] route added,network=%s,netmask=%s,nexthop=%s,iface=%s,dev=%s", network, netmask, nexthop, ipIface.Unicast, ipIface.dev.Name())
}

// SetDefaultGateway sets gw address as default gateway of ipIface
// ex) gw = "127.0.0.1"
func SetDefaultGateway(ipIface *Iface, gw string) error {

	// convert to uint32
	gwaddr, err := Str2IPAddr(gw)
	if err != nil {
		return err
	}

	RouteAdd(IPAddrAny, IPAddrAny, IPAddr(gwaddr), ipIface)
	return nil
}

// LookupTable find routing table entry whose network dst is sent
func LookupTable(dst IPAddr) (IPRoute, error) {

	var candidate *IPRoute

	// search routing table
	for _, route := range routes {

		// check if dst is the subnet of the route
		if uint32(dst)&uint32(route.netmask) == uint32(route.network) {

			// longest match
			if candidate == nil || utils.Ntoh32(uint32(candidate.netmask)) < utils.Ntoh32(uint32(route.netmask)) {
				candidate = &route
			}
		}
	}

	if candidate == nil {
		return IPRoute{}, fmt.Errorf("routing table entry not found(dst=%s)", dst)
	}
	return *candidate, nil
}
