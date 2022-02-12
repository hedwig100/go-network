package net_test

import (
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"

	"github.com/hedwig100/go-network/net"
)

func TestUDP(t *testing.T) {

	// catch CTRL+C
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	var err error

	// devices
	ether, err := net.EtherInit("tap0")
	if err != nil {
		t.Fatal(err)
	}

	// iface
	iface0, err := net.NewIPIface(etherTapIPAddr, etherTapNetmask)
	if err != nil {
		t.Fatal(err)
	}
	net.IPIfaceRegister(ether, iface0)

	// default gateway
	err = net.SetDefaultGateway(iface0, defaultGateway)
	if err != nil {
		t.Error(err)
	}

	err = net.NetInit()
	if err != nil {
		t.Error(err)
	}

	net.NetRun()

	var seq int
	src, _ := net.Str2UDPEndpoint("192.0.2.2:80")
	dst, _ := net.Str2UDPEndpoint("8.8.8.8:80")

	func() {
		for {

			// finish if interrupted
			select {
			case <-sig:
				return
			default:
			}

			time.Sleep(time.Second)
			err = net.TxHandlerUDP(src, dst, testdata)
			seq++
			if seq > 1 && err != nil { // when seq=1(first time),we get cache not found error. this is not the error
				t.Error(err)
			}
		}
	}()

	err = net.NetShutdown()
	if err != nil {
		t.Error(err)
	}
}
