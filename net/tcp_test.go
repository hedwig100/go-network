package net_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/hedwig100/go-network/net"
)

/*

1)
nc -nv -l 192.0.2.1 8080&
go test -v -run TestTCPActive > log

2)
nc -nv -l 192.0.2.1 8080&
go test -v -run TestTCPSend > log

3)
go test -v -run TestTCPPassive > log&
nc -nv 192.0.2.2 8080

4)
go test -v -run TestTCPReceive > log&
nc -nv 192.0.2.2 8080
hoge
...

*/
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
	dst, _ := net.Str2TCPEndpoint("192.0.2.1:8080")

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

	cnt := 1
	maxSendTime := 5
	for {
		if cnt == 0 && maxSendTime == 0 {
			break
		}
		if cnt == 0 && soc.Status() == net.TCPpcbStateEstablished {
			go soc.Send(errChSend, []byte(fmt.Sprintf("TCP connection%d !!!!\n", maxSendTime)))
			cnt++
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
			maxSendTime--
			if err != nil {
				t.Error("send: ", err.Error())
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

func TestTCPPassiveOpen(t *testing.T) {
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

	soc, err := net.NewTCPpcb(src)
	if err != nil {
		t.Fatal(err)
	}

	errChOpen := make(chan error)
	errChClose := make(chan error)
	go soc.Open(errChOpen, net.TCPEndpoint{}, false, 5*time.Minute)

	cnt := 1
	for {
		if cnt == 0 && soc.Status() == net.TCPpcbStateClosed {
			break
		}
		if cnt == 0 && soc.Status() == net.TCPpcbStateCloseWait {
			go soc.Close(errChClose)
			cnt++
		}

		select {
		case err = <-errChOpen:
			cnt--
			if err != nil {
				t.Error(err)
			} else {
				t.Log("open suceeded")
			}
		case err = <-errChClose:
			cnt--
			if err != nil {
				t.Error(err)
			} else {
				t.Log("close suceeded")
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

func TestTCPReceive(t *testing.T) {
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

	soc, err := net.NewTCPpcb(src)
	if err != nil {
		t.Fatal(err)
	}

	errChOpen := make(chan error)
	errChRcv := make(chan error)
	buf := make([]byte, 20)
	var n int

	go soc.Open(errChOpen, net.TCPEndpoint{}, false, 5*time.Minute)
	time.Sleep(time.Millisecond)

	cnt := 1
	maxRcvTime := 5

	for {
		if cnt == 0 && maxRcvTime == 0 {
			break
		}
		if cnt == 0 && soc.Status() == net.TCPpcbStateEstablished {
			go soc.Receive(errChRcv, buf, &n)
			cnt++
		}

		select {
		case err = <-errChOpen:
			cnt--
			if err != nil {
				t.Error(err)
			} else {
				t.Log("open suceeded")
			}
		case err = <-errChRcv:
			cnt--
			maxRcvTime--
			if err != nil {
				t.Error("receive: ", err.Error())
			} else {
				t.Log(string(buf[:n]))
				t.Log("receive suceeded")
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
