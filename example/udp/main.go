package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/hedwig100/go-network/net"
)

const (
	srcAddr = "192.0.2.2:8080"
)

func main() {
	// for logging
	file, err := os.Create("log")
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	log.SetOutput(file)

	err = net.NetInit(true)
	if err != nil {
		log.Println(err.Error())
		return
	}

	src, _ := net.Str2UDPEndpoint(srcAddr)

	net.NetRun()
	soc := net.OpenUDP()
	err = soc.Bind(src)
	if err != nil {
		log.Println(err.Error())
		return
	}

	/*

		UDP echo server

		At first, command "go run main.go&" and start the server in background.
		This server runs in 30s, if you command "nc -u 192.0.2.2 8080" an
		, you can see the text you sent will be returned.

	*/
	clock := time.After(30 * time.Second)

	func() {
		for {

			// listen
			n, data, endpoint := soc.Listen(false)
			if n > 0 {
				log.Printf("data size: %d,data: %s,endpoint: %s", n, string(data), endpoint)
				soc.Send(data, endpoint)
			}

			select {
			case <-clock:
				return
			default:
				time.Sleep(time.Millisecond)
			}
		}
	}()

	err = net.Close(soc)
	if err != nil {
		log.Println(err.Error())
	}

	err = net.NetShutdown()
	if err != nil {
		log.Println(err.Error())
	}
}
