package arp

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/hedwig100/go-network/devices"
	"github.com/hedwig100/go-network/ip"
)

const (
	ArpCacheSize uint8 = 32

	// cache state
	ArpCacheStateFree       uint8 = 0
	ArpCacheStateImcomplete uint8 = 1
	ArpCacheStateResolved   uint8 = 2
	ArpCacheStateStatic     uint8 = 3
)

var mutex sync.Mutex
var caches [ArpCacheSize]arpCacheEntry

// arpCacheEntry is arp cache table's entry
type arpCacheEntry struct {

	// cache state
	state uint8

	// protocol address
	pa ip.IPAddr

	// hardware address
	ha devices.EthernetAddress

	// time
	timeval time.Time
}

// arpCacheAlloc searches empty cache entry in the cache table and returns the index,
// if no empty entry is found, index of the oldest entry is returned.
func arpCacheAlloc() int {

	oldestIndex := -1
	var oldest arpCacheEntry

	for i, cache := range caches {

		// empty cache
		if cache.state == ArpCacheStateFree {
			return i
		}

		// update if cache's timeval is older than oldest's timeval
		if oldestIndex < 0 || oldest.timeval.Before(cache.timeval) {
			oldestIndex = i
			oldest = cache
		}
	}

	return oldestIndex
}

// arpCacheInsert inserts cache entry to the cache table
func arpCacheInsert(pa ip.IPAddr, ha devices.EthernetAddress) {

	index := arpCacheAlloc()
	timeval := time.Now()
	caches[index] = arpCacheEntry{
		state:   ArpCacheStateResolved,
		pa:      pa,
		ha:      ha,
		timeval: timeval,
	}
	log.Printf("[D] ARP cache insert pa=%s,ha=%s,timeval=%s", pa, ha, timeval)

}

// arpCacheSelect selects cache entry from the cache table
// and returns index of the entry
func arpCacheSelect(pa ip.IPAddr) (int, error) {

	for i, cache := range caches {
		if cache.state != ArpCacheStateFree && cache.pa == pa {
			return i, nil
		}
	}

	return 0, fmt.Errorf("cache not found(pa=%s)", pa)
}

// arpCacheUpdate updates cache entry in the cache table
func arpCacheUpdate(pa ip.IPAddr, ha devices.EthernetAddress) error {

	// get cache index
	index, err := arpCacheSelect(pa)
	if err != nil {
		return err
	}

	// update
	timeval := time.Now()
	caches[index] = arpCacheEntry{
		state:   ArpCacheStateResolved,
		pa:      pa,
		ha:      ha,
		timeval: timeval,
	}
	log.Printf("[D] ARP cache update ps=%s,ha=%s,timeval=%s", pa, ha, timeval)
	return nil
}

// arpCacheDelete deletes cache entry from the cache table
func arpCacheDelete(index int) error {
	if index < 0 || index >= int(ArpCacheSize) {
		return fmt.Errorf("cache table index out of range")
	}

	caches[index] = arpCacheEntry{
		state: ArpCacheStateFree,
	}
	return nil
}
