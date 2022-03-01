package icmp_test

import (
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"

	"github.com/hedwig100/go-network/pkg"
	"github.com/hedwig100/go-network/pkg/device"
	"github.com/hedwig100/go-network/pkg/icmp"
	"github.com/hedwig100/go-network/pkg/ip"
)

const (
	loopbackIPAddr  = "127.0.0.1"
	loopbackNetmask = "255.0.0.0"

	etherTapIPAddr  = "192.0.2.2"
	etherTapNetmask = "255.255.255.0"

	defaultGateway = "192.0.2.1"
)

var testdata = []byte{
	0x45, 0x00, 0x00, 0x30,
	0x00, 0x80, 0x00, 0x00,
	0xff, 0x01, 0xbd, 0x4a,
	0x7f, 0x00, 0x00, 0x01,
	0x7f, 0x00, 0x00, 0x01,
	0x08, 0x00, 0x35, 0x64,
	0x00, 0x80, 0x00, 0x01,
	0x31, 0x32, 0x33, 0x34,
	0x35, 0x36, 0x37, 0x38,
	0x39, 0x30, 0x21, 0x40,
	0x23, 0x24, 0x25, 0x5e,
	0x26, 0x2a, 0x28, 0x29,
}

/*

go test -v ./pkg/icmp/ -run TestICMP > log&
ping 192.0.2.2
...

*/

func TestICMP(t *testing.T) {

	// catch CTRL+C
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	var err error

	// devices
	_ = device.NullInit("null0")
	loop := device.LoopbackInit("loop0")
	ether, err := device.EtherInit("tap0")
	if err != nil {
		t.Fatal(err)
	}

	// iface
	iface0, err := ip.NewIface(loopbackIPAddr, loopbackNetmask)
	if err != nil {
		t.Fatal(err)
	}
	ip.IfaceRegister(loop, iface0)

	iface1, err := ip.NewIface(etherTapIPAddr, etherTapNetmask)
	if err != nil {
		t.Fatal(err)
	}
	ip.IfaceRegister(ether, iface1)

	// default gateway
	err = ip.SetDefaultGateway(iface1, defaultGateway)
	if err != nil {
		t.Error(err)
	}

	err = pkg.NetInit(false)
	if err != nil {
		t.Error(err)
	}

	pkg.NetRun()

	src := iface1.Unicast
	dst, _ := ip.Str2IPAddr("8.8.8.8") //net.Str2IPAddr(defaultGateway)
	id := uint32(109)
	seq := uint32(0)

	func() {
		for {

			// finish if interrupted
			select {
			case <-sig:
				return
			default:
			}

			time.Sleep(time.Second)
			err = icmp.TxHandlerICMP(icmp.ICMPTypeEcho, 0, (id<<16 | seq), testdata, src, ip.IPAddr(dst))
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
