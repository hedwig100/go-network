package ip_test

import (
	"testing"
	"time"

	"github.com/hedwig100/go-network/pkg"
	"github.com/hedwig100/go-network/pkg/device"
	"github.com/hedwig100/go-network/pkg/ip"
	"github.com/hedwig100/go-network/pkg/net"
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

func TestIP(t *testing.T) {
	var err error

	// devices
	_ = device.NullInit("null0")
	loop := device.LoopbackInit("loop0")
	ether, err := device.EtherInit("tap0")
	if err != nil {
		t.Error(err)
	}

	// iface
	iface0, err := ip.NewIface(loopbackIPAddr, loopbackNetmask)
	if err != nil {
		t.Error(err)
	}
	ip.IfaceRegister(loop, iface0)

	iface1, err := ip.NewIface(etherTapIPAddr, etherTapNetmask)
	if err != nil {
		t.Error(err)
	}
	ip.IfaceRegister(ether, iface1)

	// default gateway
	err = ip.SetDefaultGateway(iface0, defaultGateway)
	if err != nil {
		return
	}

	err = pkg.NetInit(false)
	if err != nil {
		t.Error(err)
	}

	for i := 0; i < 5; i++ {
		time.Sleep(time.Millisecond)
		err = net.DeviceOutput(ether, testdata, net.ProtoTypeIP, device.EtherAddrAny)
		if err != nil {
			t.Error(err)
		}
	}

	err = pkg.NetShutdown()
	if err != nil {
		t.Error(err)
	}
}
