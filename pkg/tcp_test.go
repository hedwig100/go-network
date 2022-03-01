package pkg_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/hedwig100/go-network/pkg"
	"github.com/hedwig100/go-network/pkg/device"
	"github.com/hedwig100/go-network/pkg/ip"
)

/*

1)
nc -nv -l 192.0.2.1 8080&
go test -v ./pkg/ -run TestTCPActive > log

2)
nc -nv -l 192.0.2.1 8080&
go test -v ./pkg/ -run TestTCPSend > log

3)
go test -v ./pkg/ -run TestTCPPassive > log&
nc -nv 192.0.2.2 8080

4)
go test -v ./pkg/ -run TestTCPReceive > log&
nc -nv 192.0.2.2 8080
hoge
...

*/
// TODO: Close doesn't succeed (FIN segment doesn't reach?)

const (
	loopbackIPAddr  = "127.0.0.1"
	loopbackNetmask = "255.0.0.0"

	etherTapIPAddr  = "192.0.2.2"
	etherTapNetmask = "255.255.255.0"

	defaultGateway = "192.0.2.1"
)

func TestTCPActiveOpenClose(t *testing.T) {
	var err error

	// devices
	_ = device.NullInit("null0")
	loop := device.LoopbackInit("loop0")
	ether, err := device.EtherInit("tap0")
	if err != nil {
		t.Fatal(err)
	}

	// iface
	iface0, err := ip.NewIPIface(loopbackIPAddr, loopbackNetmask)
	if err != nil {
		t.Fatal(err)
	}
	ip.IPIfaceRegister(loop, iface0)

	iface1, err := ip.NewIPIface(etherTapIPAddr, etherTapNetmask)
	if err != nil {
		t.Fatal(err)
	}
	ip.IPIfaceRegister(ether, iface1)

	// default gateway
	err = ip.SetDefaultGateway(iface1, defaultGateway)
	if err != nil {
		t.Fatal(err)
	}

	err = pkg.NetInit(false)
	if err != nil {
		t.Fatal(err)
	}

	pkg.NetRun()

	src, _ := pkg.Str2TCPEndpoint("192.0.2.2:8080")
	dst, _ := pkg.Str2TCPEndpoint("192.0.2.1:8080")

	soc, err := pkg.NewTCPpcb(src)
	if err != nil {
		t.Fatal(err)
	}

	errChOpen := make(chan error)
	errChClose := make(chan error)

	go soc.Open(errChOpen, dst, true, 5*time.Minute)
	open := 1
	close := 0

	for {
		if open == 0 && close == 0 && soc.Status() == pkg.TCPpcbStateEstablished {
			go soc.Close(errChClose)
			close++
		}
		if open == 0 && close == 0 {
			break
		}

		select {
		case err = <-errChOpen:
			open--
			if err != nil {
				t.Error(err)
			} else {
				t.Log("open suceeded")
			}
		case err = <-errChClose:
			close--
			if err != nil {
				t.Error(err)
			} else {
				t.Log("close suceeded")
			}
		}
	}

	err = pkg.NetShutdown()
	if err != nil {
		t.Error(err)
	}

}

func TestTCPSend(t *testing.T) {
	var err error

	// devices
	_ = device.NullInit("null0")
	loop := device.LoopbackInit("loop0")
	ether, err := device.EtherInit("tap0")
	if err != nil {
		t.Fatal(err)
	}

	// iface
	iface0, err := ip.NewIPIface(loopbackIPAddr, loopbackNetmask)
	if err != nil {
		t.Fatal(err)
	}
	ip.IPIfaceRegister(loop, iface0)

	iface1, err := ip.NewIPIface(etherTapIPAddr, etherTapNetmask)
	if err != nil {
		t.Fatal(err)
	}
	ip.IPIfaceRegister(ether, iface1)

	// default gateway
	err = ip.SetDefaultGateway(iface1, defaultGateway)
	if err != nil {
		t.Fatal(err)
	}

	err = pkg.NetInit(false)
	if err != nil {
		t.Fatal(err)
	}

	pkg.NetRun()

	src, _ := pkg.Str2TCPEndpoint("192.0.2.2:8080")
	dst, _ := pkg.Str2TCPEndpoint("192.0.2.1:8080")

	soc, err := pkg.NewTCPpcb(src)
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
		if cnt == 0 && soc.Status() == pkg.TCPpcbStateEstablished {
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

	err = pkg.NetShutdown()
	if err != nil {
		t.Error(err)
	}

}

func TestTCPPassiveOpen(t *testing.T) {
	var err error

	// devices
	_ = device.NullInit("null0")
	loop := device.LoopbackInit("loop0")
	ether, err := device.EtherInit("tap0")
	if err != nil {
		t.Fatal(err)
	}

	// iface
	iface0, err := ip.NewIPIface(loopbackIPAddr, loopbackNetmask)
	if err != nil {
		t.Fatal(err)
	}
	ip.IPIfaceRegister(loop, iface0)

	iface1, err := ip.NewIPIface(etherTapIPAddr, etherTapNetmask)
	if err != nil {
		t.Fatal(err)
	}
	ip.IPIfaceRegister(ether, iface1)

	// default gateway
	err = ip.SetDefaultGateway(iface1, defaultGateway)
	if err != nil {
		t.Fatal(err)
	}

	err = pkg.NetInit(false)
	if err != nil {
		t.Fatal(err)
	}

	pkg.NetRun()

	src, _ := pkg.Str2TCPEndpoint("192.0.2.2:8080")

	soc, err := pkg.NewTCPpcb(src)
	if err != nil {
		t.Fatal(err)
	}

	errChOpen := make(chan error)
	errChClose := make(chan error)
	go soc.Open(errChOpen, pkg.TCPEndpoint{}, false, 5*time.Minute)

	cnt := 1
	for {
		if cnt == 0 && soc.Status() == pkg.TCPpcbStateClosed {
			break
		}
		if cnt == 0 && soc.Status() == pkg.TCPpcbStateCloseWait {
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
			if err != nil && err.Error() != "connection closed" { // passive close
				t.Error(err)
			} else {
				t.Log("close suceeded")
			}
		default:
			time.Sleep(time.Millisecond)
		}
	}

	err = pkg.NetShutdown()
	if err != nil {
		t.Error(err)
	}

}

func TestTCPReceive(t *testing.T) {
	var err error

	// devices
	_ = device.NullInit("null0")
	loop := device.LoopbackInit("loop0")
	ether, err := device.EtherInit("tap0")
	if err != nil {
		t.Fatal(err)
	}

	// iface
	iface0, err := ip.NewIPIface(loopbackIPAddr, loopbackNetmask)
	if err != nil {
		t.Fatal(err)
	}
	ip.IPIfaceRegister(loop, iface0)

	iface1, err := ip.NewIPIface(etherTapIPAddr, etherTapNetmask)
	if err != nil {
		t.Fatal(err)
	}
	ip.IPIfaceRegister(ether, iface1)

	// default gateway
	err = ip.SetDefaultGateway(iface1, defaultGateway)
	if err != nil {
		t.Fatal(err)
	}

	err = pkg.NetInit(false)
	if err != nil {
		t.Fatal(err)
	}

	pkg.NetRun()

	src, _ := pkg.Str2TCPEndpoint("192.0.2.2:8080")

	soc, err := pkg.NewTCPpcb(src)
	if err != nil {
		t.Fatal(err)
	}

	errChOpen := make(chan error)
	errChRcv := make(chan error)
	buf := make([]byte, 20)
	var n int

	go soc.Open(errChOpen, pkg.TCPEndpoint{}, false, 5*time.Minute)
	time.Sleep(time.Millisecond)

	cnt := 1
	maxRcvTime := 5

	for {
		if cnt == 0 && maxRcvTime == 0 {
			break
		}
		if cnt == 0 && soc.Status() == pkg.TCPpcbStateEstablished {
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

	err = pkg.NetShutdown()
	if err != nil {
		t.Error(err)
	}

}
