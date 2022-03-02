package net

import (
	"fmt"
	"log"
)

// Open activate the receive handler of the device and
// activate the receive handler of the protocol
func Open(done chan struct{}) {
	// activate the receive handler of the device
	for _, dev := range devices {
		go dev.RxHandler(done)
	}

	// activate the receive handler of the protocol
	for i, proto := range protos {
		go proto.RxHandler(protoBuffers[i], done)
	}
}

// Close closes all the devices
func Close() (err error) {
	for _, dev := range devices {

		if !isUp(dev) {
			return fmt.Errorf("already closed dev=%s", dev.Name())
		}

		// close the channel and stop the receive handler
		err = dev.Close()
		if err != nil {
			return
		}
		log.Printf("[I] close device dev=%s", dev.Name())
	}
	return
}
