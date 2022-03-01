package net_test

import (
	"testing"
	"time"

	"github.com/hedwig100/go-network/net"
	"github.com/hedwig100/go-network/net/device"
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
	iface0, err := net.NewIPIface(loopbackIPAddr, loopbackNetmask)
	if err != nil {
		t.Error(err)
	}
	net.IPIfaceRegister(loop, iface0)

	iface1, err := net.NewIPIface(etherTapIPAddr, etherTapNetmask)
	if err != nil {
		t.Error(err)
	}
	net.IPIfaceRegister(ether, iface1)

	// default gateway
	err = net.SetDefaultGateway(iface0, defaultGateway)
	if err != nil {
		return
	}

	err = net.NetInit(false)
	if err != nil {
		t.Error(err)
	}

	for i := 0; i < 5; i++ {
		time.Sleep(time.Millisecond)
		err = device.DeviceOutput(ether, testdata, device.ProtocolTypeIP, device.EtherAddrAny)
		if err != nil {
			t.Error(err)
		}
	}

	err = net.NetShutdown()
	if err != nil {
		t.Error(err)
	}
}
