package net_test

import (
	"testing"
	"time"

	"github.com/hedwig100/go-network/net"
)

func TestNull(t *testing.T) {
	t.Skip()
	var err error

	dev := net.NullInit("null0")

	err = net.NetInit()
	if err != nil {
		t.Error(err)
	}

	for i := 0; i < 5; i++ {
		time.Sleep(time.Millisecond)
		err = net.DeviceOutput(dev, []byte{0xff, 0xff, 0x11}, 0x0000, net.EtherAddrBroadcast)
		if err != nil {
			t.Error(err)
		}
	}

	err = net.NetShutdown()
	if err != nil {
		t.Error(err)
	}
}

func TestLoopback(t *testing.T) {
	t.Skip()
	var err error

	dev := net.LoopbackInit("loopback0")

	err = net.NetInit()
	if err != nil {
		t.Error(err)
	}

	for i := 0; i < 5; i++ {
		time.Sleep(time.Millisecond)
		err = net.DeviceOutput(dev, []byte{0xff, 0xff, 0x11}, 0x0000, net.EtherAddrBroadcast)
		if err != nil {
			t.Error(err)
		}
	}

	err = net.NetShutdown()
	if err != nil {
		t.Error(err)
	}
}

func TestEther(t *testing.T) {
	var err error

	dev, err := net.EtherInit("tap0")
	if err != nil {
		t.Error(err)
	}

	err = net.NetInit()
	if err != nil {
		t.Error(err)
	}

	for i := 0; i < 5; i++ {
		time.Sleep(time.Second)
		err = net.DeviceOutput(dev, []byte{0xff, 0xff, 0x11}, 0x0000, net.EtherAddrBroadcast)
		if err != nil {
			t.Error(err)
		}
	}

	err = net.NetShutdown()
	if err != nil {
		t.Error(err)
	}
}
