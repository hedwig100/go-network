package udp_test

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"

	"github.com/hedwig100/go-network/pkg"
	"github.com/hedwig100/go-network/pkg/device"
	"github.com/hedwig100/go-network/pkg/ip"
	"github.com/hedwig100/go-network/pkg/udp"
)

/*

1)
go test -v ./pkg/udp -run TestSendUDP > log&
nc -u -l 192.0.2.1 80

2)
go test -v ./pkg/udp -run TestSocketUDP > log&
nc -u 192.0.2.2 7
hoge (followed by the same reply)
...

*/

const (
	etherTapIPAddr  = "192.0.2.2"
	etherTapNetmask = "255.255.255.0"

	defaultGateway = "192.0.2.1"
)

func TestUDP(t *testing.T) {

	// catch CTRL+C
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	var err error

	// devices
	ether, err := device.EtherInit("tap0")
	if err != nil {
		t.Fatal(err)
	}

	// iface
	iface0, err := ip.NewIface(etherTapIPAddr, etherTapNetmask)
	if err != nil {
		t.Fatal(err)
	}
	ip.IfaceRegister(ether, iface0)

	// default gateway
	err = ip.SetDefaultGateway(iface0, defaultGateway)
	if err != nil {
		t.Error(err)
	}

	err = pkg.NetInit(false)
	if err != nil {
		t.Error(err)
	}

	pkg.NetRun()

	var seq int
	src, _ := udp.Str2Endpoint("192.0.2.2:80")
	dst, _ := udp.Str2Endpoint("8.8.8.8:80")

	func() {
		for {

			// finish if interrupted
			select {
			case <-sig:
				return
			default:
			}

			time.Sleep(time.Second)
			err = udp.TxHandler(src, dst, []byte("hello"))
			seq++
			if seq > 1 && err != nil { // when seq=1(first time),we get cache not found error. this is not the error
				t.Error(err)
			}
		}
	}()

	err = pkg.NetShutdown()
	if err != nil {
		t.Error(err)
	}
}

func TestSendUDP(t *testing.T) {

	// catch CTRL+C
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	var err error

	// devices
	ether, err := device.EtherInit("tap0")
	if err != nil {
		t.Fatal(err)
	}

	// iface
	iface0, err := ip.NewIface(etherTapIPAddr, etherTapNetmask)
	if err != nil {
		t.Fatal(err)
	}
	ip.IfaceRegister(ether, iface0)

	// default gateway
	err = ip.SetDefaultGateway(iface0, defaultGateway)
	if err != nil {
		t.Error(err)
	}

	err = pkg.NetInit(false)
	if err != nil {
		t.Error(err)
	}

	pkg.NetRun()

	var seq int
	src, _ := udp.Str2Endpoint("192.0.2.2:80")
	dst, _ := udp.Str2Endpoint("192.0.2.1:80")

	func() {
		for {

			// finish if interrupted
			select {
			case <-sig:
				return
			default:
			}

			time.Sleep(time.Second)
			err = udp.TxHandler(src, dst, []byte("hello world!\n"))
			seq++
			if seq > 1 && err != nil { // when seq=1(first time),we get cache not found error. this is not the error
				t.Error(err)
			}
		}
	}()

	err = pkg.NetShutdown()
	if err != nil {
		t.Error(err)
	}
}

func TestSocketUDP(t *testing.T) {
	// catch CTRL+C
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	var err error

	// devices
	ether, err := device.EtherInit("tap0")
	if err != nil {
		t.Fatal(err)
	}

	// iface
	iface0, err := ip.NewIface(etherTapIPAddr, etherTapNetmask)
	if err != nil {
		t.Fatal(err)
	}
	ip.IfaceRegister(ether, iface0)

	// default gateway
	err = ip.SetDefaultGateway(iface0, defaultGateway)
	if err != nil {
		t.Error(err)
	}

	err = pkg.NetInit(false)
	if err != nil {
		t.Error(err)
	}

	pkg.NetRun()

	// var seq int
	src, _ := udp.Str2Endpoint("192.0.2.2:7")
	// dst, _ := pkg.Str2UDPEndpoint("192.0.2.1:7")

	sock := udp.OpenUDP()
	err = sock.Bind(src)
	if err != nil {
		t.Error(err)
	}

	func() {
		for {

			// finish if interrupted
			select {
			case <-sig:
				return
			default:
			}

			time.Sleep(time.Second)

			// send
			// err = sock.Send([]byte("hello world!"), dst)
			// seq++
			// if seq > 1 && err != nil { // when seq=1(first time),we get cache not found error. this is not the error
			// 	t.Error(err)
			// }

			// listen
			n, data, endpoint := sock.Listen(false)
			if n > 0 {
				log.Printf("data size: %d,data: %s,endpoint: %s", n, string(data), endpoint)
				sock.Send(data, endpoint)
			}
		}
	}()

	err = udp.Close(sock)
	if err != nil {
		t.Error(err)
	}
	err = pkg.NetShutdown()
	if err != nil {
		t.Error(err)
	}

}
