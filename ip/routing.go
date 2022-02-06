package ip

import (
	"fmt"
	"log"

	"github.com/hedwig100/go-network/utils"
)

var routes []IPRoute

// IPRoute is routing table entry
type IPRoute struct {
	network IPAddr
	netmask IPAddr
	nexthop IPAddr
	ipIface *IPIface
}

// IPRouteeAdd add routing table entry to routing table
func IPRouteAdd(network IPAddr, netmask IPAddr, nexthop IPAddr, ipIface *IPIface) {
	routes = append(routes, IPRoute{
		network: network,
		netmask: netmask,
		nexthop: nexthop,
		ipIface: ipIface,
	})
	log.Printf("[I] route added,netword=%s,netmask=%s,nexthop=%s,iface=%s,dev=%s", network, netmask, nexthop, ipIface.unicast, ipIface.dev.Name())
}

// SetDefaultGateway sets gw address as default gateway of ipIface
// ex) gw = "127.0.0.1"
func SetDefaultGateway(ipIface *IPIface, gw string) error {

	// convert to uint32
	gwaddr, err := str2IPAddr(gw)
	if err != nil {
		return err
	}

	IPRouteAdd(IPAddrAny, IPAddrAny, IPAddr(gwaddr), ipIface)
	return nil
}

// LookupTable find routing table entry whose network dst is sent
func LookupTable(dst IPAddr) (IPRoute, error) {

	var candidate *IPRoute

	// search routing table
	for _, route := range routes {
		if uint32(dst)&uint32(route.netmask) == uint32(route.netmask) {

			// longest match
			if candidate != nil || utils.NtoH32(uint32(candidate.netmask)) < utils.NtoH32(uint32(route.netmask)) {
				candidate = &route
			}
		}
	}

	if candidate == nil {
		return IPRoute{}, fmt.Errorf("routing table entry not found(dst=%s)", dst)
	}
	return *candidate, nil
}
