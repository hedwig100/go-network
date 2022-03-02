package ip

import (
	"fmt"
	"log"

	"github.com/hedwig100/go-network/pkg/utils"
)

var routes []route

// route is routing table entry
type route struct {
	network Addr
	netmask Addr
	nexthop Addr
	Iface   *Iface
}

// routeAdd add routing table entry to routing table
func routeAdd(network Addr, netmask Addr, nexthop Addr, iface *Iface) {
	routes = append(routes, route{
		network: network,
		netmask: netmask,
		nexthop: nexthop,
		Iface:   iface,
	})
	log.Printf("[I] route added,network=%s,netmask=%s,nexthop=%s,iface=%s,dev=%s", network, netmask, nexthop, iface.Unicast, iface.dev.Name())
}

// SetDefaultGateway sets gw address as default gateway of ipIface
// ex) gw = "127.0.0.1"
func SetDefaultGateway(ipIface *Iface, gw string) error {

	// convert to uint32
	gwaddr, err := Str2Addr(gw)
	if err != nil {
		return err
	}

	routeAdd(AddrAny, AddrAny, Addr(gwaddr), ipIface)
	return nil
}

// LookupTable find routing table entry whose network dst is sent
func LookupTable(dst Addr) (route, error) {

	var candidate *route

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
		return route{}, fmt.Errorf("routing table entry not found(dst=%s)", dst)
	}
	return *candidate, nil
}
