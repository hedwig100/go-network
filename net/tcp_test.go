package net_test

import (
	"testing"
	"time"

	"github.com/hedwig100/go-network/net"
)

// TODO: Close doesn't succeed (FIN segment doesn't reach?)
func TestTCPActiveOpenClose(t *testing.T) {
	var err error

	// devices
	_ = net.NullInit("null0")
	loop := net.LoopbackInit("loop0")
	ether, err := net.EtherInit("tap0")
	if err != nil {
		t.Fatal(err)
	}

	// iface
	iface0, err := net.NewIPIface(loopbackIPAddr, loopbackNetmask)
	if err != nil {
		t.Fatal(err)
	}
	net.IPIfaceRegister(loop, iface0)

	iface1, err := net.NewIPIface(etherTapIPAddr, etherTapNetmask)
	if err != nil {
		t.Fatal(err)
	}
	net.IPIfaceRegister(ether, iface1)

	// default gateway
	err = net.SetDefaultGateway(iface1, defaultGateway)
	if err != nil {
		t.Fatal(err)
	}

	err = net.NetInit()
	if err != nil {
		t.Fatal(err)
	}

	net.NetRun()

	src, _ := net.Str2TCPEndpoint("192.0.2.2:8080")
	dst, _ := net.Str2TCPEndpoint("192.0.2.1:100")

	soc, err := net.NewTCPpcb(src)
	if err != nil {
		t.Fatal(err)
	}

	errChOpen := make(chan error)
	errChClose := make(chan error)

	go soc.Open(errChOpen, dst, true, 5*time.Minute)
	time.Sleep(10 * time.Second)
	go soc.Close(errChClose)

	err = <-errChOpen
	if err != nil {
		t.Error(err)
	} else {
		t.Log("open suceeded")
	}

	err = <-errChClose
	if err != nil {
		t.Error(err)
	} else {
		t.Log("close suceeded")
	}

	err = net.NetShutdown()
	if err != nil {
		t.Error(err)
	}

}

func TestTCPSend(t *testing.T) {
	var err error

	// devices
	_ = net.NullInit("null0")
	loop := net.LoopbackInit("loop0")
	ether, err := net.EtherInit("tap0")
	if err != nil {
		t.Fatal(err)
	}

	// iface
	iface0, err := net.NewIPIface(loopbackIPAddr, loopbackNetmask)
	if err != nil {
		t.Fatal(err)
	}
	net.IPIfaceRegister(loop, iface0)

	iface1, err := net.NewIPIface(etherTapIPAddr, etherTapNetmask)
	if err != nil {
		t.Fatal(err)
	}
	net.IPIfaceRegister(ether, iface1)

	// default gateway
	err = net.SetDefaultGateway(iface1, defaultGateway)
	if err != nil {
		t.Fatal(err)
	}

	err = net.NetInit()
	if err != nil {
		t.Fatal(err)
	}

	net.NetRun()

	src, _ := net.Str2TCPEndpoint("192.0.2.2:8080")
	dst, _ := net.Str2TCPEndpoint("192.0.2.1:8080")

	soc, err := net.NewTCPpcb(src)
	if err != nil {
		t.Fatal(err)
	}

	errChOpen := make(chan error)
	errChSend := make(chan error)

	go soc.Open(errChOpen, dst, true, 5*time.Minute)
	time.Sleep(2 * time.Second)
	go soc.Send(errChSend, []byte("TCP connection!!"))

	cnt := 2
	for {
		if cnt == 0 {
			break
		}

		select {
		case err = <-errChOpen:
			cnt--
			if err != nil {
				t.Error(err)
			} else {
				t.Log("open suceeded")
			}
		case err = <-errChSend:
			cnt--
			if err != nil {
				t.Error(err)
			} else {
				t.Log("send suceeded")
			}
		default:
			time.Sleep(time.Millisecond)
		}
	}

	err = net.NetShutdown()
	if err != nil {
		t.Error(err)
	}

}
