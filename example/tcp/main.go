package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/hedwig100/go-network/pkg"
	"github.com/hedwig100/go-network/pkg/tcp"
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

	err = pkg.NetInit(true)
	if err != nil {
		log.Println(err.Error())
		return
	}

	src, _ := tcp.Str2Endpoint(srcAddr)

	pkg.NetRun()
	soc, err := tcp.Newpcb(src)
	if err != nil {
		log.Println(err.Error())
		return
	}

	/*

		TCP echo server

		At first, command "go run main.go&" and start the server in background.
		This server runs in 30s, if you command "nc -nv 192.0.2.2 8080" an
		, you can see the text you sent will be returned.

	*/
	clock := time.After(30 * time.Second)

	// for open
	errOpen := make(chan error)
	go soc.Open(errOpen, tcp.Endpoint{}, false, 5*time.Minute)

	// for receive
	errRcv := make(chan error)
	buf := make([]byte, 100)
	var n, rcv int

	// for send
	errSnd := make(chan error)

	func() {
		for {
			if rcv == 0 && soc.Status() == tcp.PCBStateEstablished {
				go soc.Receive(errRcv, buf, &n)
				rcv++
			}

			select {
			case err = <-errOpen:
				if err != nil {
					log.Println(err.Error())
					return
				} else {
					log.Println("open succeeded")
				}
			case err = <-errRcv:
				if err != nil {
					log.Println(err.Error())
					return
				} else {
					log.Printf("received: %s", string(buf[:n]))
					rcv--
					go soc.Send(errSnd, buf[:n])
				}
			case err = <-errSnd:
				if err != nil {
					log.Println(err.Error())
					return
				} else {
					log.Println("send succeeded")
				}
			case <-clock:
				return
			default:
				time.Sleep(time.Millisecond)
			}
		}
	}()

	// close
	errClose := make(chan error)
	go soc.Close(errClose)
	err = <-errClose
	if err != nil {
		log.Println(err.Error())
		return
	} else {
		log.Println("close succeeded")
	}

	err = tcp.Deletepcb(soc)
	if err != nil {
		log.Println(err.Error())
	}

	err = pkg.NetShutdown()
	if err != nil {
		log.Println(err.Error())
	}
}
