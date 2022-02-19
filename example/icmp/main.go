package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/hedwig100/go-network/net"
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
		fmt.Println(err.Error())
		return
	}

	net.NetRun()
	time.Sleep(10 * time.Second)
	/*
		In this section, you can ping to 192.0.2.2
	*/
	net.NetShutdown()

}
