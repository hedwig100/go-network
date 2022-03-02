package arp

import "time"

const cacheTimeout time.Duration = 30 * time.Second

// timer
func timer(done chan struct{}) {
	for {

		// check if process finishes or not
		select {
		case <-done:
			return
		default:
		}

		now := time.Now()
		for i, cache := range caches {
			if cache.state != cacheFree && cache.timeval.Add(cacheTimeout).Before(now) {
				cacheDelete(i) // no error
			}
		}

		// sleep for a second
		time.Sleep(time.Second)
	}
}
