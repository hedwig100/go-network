package arp

import "time"

const ArpCacheTimeout time.Duration = 30 * time.Second

// arpTimer
func arpTimer(done chan struct{}) {
	for {

		// check if process finishes or not
		select {
		case <-done:
			return
		default:
		}

		now := time.Now()
		for i, cache := range caches {
			if cache.timeval.Add(ArpCacheTimeout).After(now) {
				arpCacheDelete(i) // no error
			}
		}

		// sleep for a second
		time.Sleep(time.Second)
	}
}
