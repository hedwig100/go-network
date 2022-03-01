package pkg_test

import (
	"testing"
	"time"

	"github.com/hedwig100/go-network/pkg"
	"github.com/hedwig100/go-network/pkg/device"
	"github.com/hedwig100/go-network/pkg/net"
)

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
	iface0, err := pkg.NewIPIface(loopbackIPAddr, loopbackNetmask)
	if err != nil {
		t.Error(err)
	}
	pkg.IPIfaceRegister(loop, iface0)

	iface1, err := pkg.NewIPIface(etherTapIPAddr, etherTapNetmask)
	if err != nil {
		t.Error(err)
	}
	pkg.IPIfaceRegister(ether, iface1)

	// default gateway
	err = pkg.SetDefaultGateway(iface0, defaultGateway)
	if err != nil {
		return
	}

	err = pkg.NetInit(false)
	if err != nil {
		t.Error(err)
	}

	for i := 0; i < 5; i++ {
		time.Sleep(time.Millisecond)
		err = net.DeviceOutput(ether, testdata, net.ProtocolTypeIP, device.EtherAddrAny)
		if err != nil {
			t.Error(err)
		}
	}

	err = pkg.NetShutdown()
	if err != nil {
		t.Error(err)
	}
}
