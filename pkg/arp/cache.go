package arp

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/hedwig100/go-network/pkg/device"
	"github.com/hedwig100/go-network/pkg/ip"
)

const (
	cacheSize uint8 = 32

	// cache state
	cacheFree       uint8 = 0
	cacheImcomplete uint8 = 1
	cacheResolved   uint8 = 2
	cacheStatic     uint8 = 3
)

var (
	mutex  sync.Mutex
	caches [cacheSize]cacheEntry
)

// cacheEntry is arp cache table's entry
type cacheEntry struct {

	// cache state
	state uint8

	// protocol address
	pa ip.Addr

	// hardware address
	ha device.EtherAddr

	// time
	timeval time.Time
}

// cacheAlloc searches empty cache entry in the cache table and returns the index,
// if no empty entry is found, index of the oldest entry is returned.
func cacheAlloc() int {

	var id int
	var oldest cacheEntry

	for i, cache := range caches {

		// empty cache
		if cache.state == cacheFree {
			return i
		}

		// update if cache's timeval is older than oldest's timeval
		if oldest.timeval.After(cache.timeval) {
			id = i
			oldest = cache
		}
	}

	return id
}

// cacheInsert inserts cache entry to the cache table
func cacheInsert(pa ip.Addr, ha device.EtherAddr) {

	id := cacheAlloc()
	timeval := time.Now()
	caches[id] = cacheEntry{
		state:   cacheResolved,
		pa:      pa,
		ha:      ha,
		timeval: timeval,
	}
	log.Printf("[D] ARP cache insert pa=%s,ha=%s,timeval=%s", pa, ha, timeval)

}

// cacheSelect selects cache entry from the cache table
// and returns index of the entry
func cacheSelect(pa ip.Addr) (int, error) {

	for i, cache := range caches {
		if cache.state != cacheFree && cache.pa == pa {
			return i, nil
		}
	}

	return 0, fmt.Errorf("cache not found(pa=%s)", pa)
}

// cacheUpdate updates cache entry in the cache table
// return true if cache was inserted before and update is successful
// return false if cache was not there and update is unsuccessful
func cacheUpdate(pa ip.Addr, ha device.EtherAddr) bool {

	// get cache index
	id, err := cacheSelect(pa)
	if err != nil {
		return false
	}

	// update
	timeval := time.Now()
	caches[id] = cacheEntry{
		state:   cacheResolved,
		pa:      pa,
		ha:      ha,
		timeval: timeval,
	}
	log.Printf("[D] ARP cache update ps=%s,ha=%s,timeval=%s", pa, ha, timeval)
	return true
}

// cacheDelete deletes cache entry from the cache table
func cacheDelete(id int) error {
	if id < 0 || id >= int(cacheSize) {
		return fmt.Errorf("cache table index out of range")
	}

	log.Printf("[D] ARP cache delete ps=%s,ha=%s", caches[id].pa, caches[id].ha)
	caches[id] = cacheEntry{
		state: cacheFree,
	}
	return nil
}
