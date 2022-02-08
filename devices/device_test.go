package devices_test

import (
	"testing"
	"time"

	"github.com/hedwig100/go-network/devices"
	"github.com/hedwig100/go-network/net"
)

func TestNull(t *testing.T) {
	t.Skip()
	var err error

	dev, err := devices.NullInit("null0")
	if err != nil {
		t.Error(err)
	}

	err = net.NetRun()
	if err != nil {
		t.Error(err)
	}

	for i := 0; i < 5; i++ {
		time.Sleep(time.Millisecond)
		err = net.DeviceOutput(dev, []byte{0xff, 0xff, 0x11}, 0x0000, devices.EtherAddrBroadcast)
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

	dev, err := devices.LoopbackInit("loopback0")
	if err != nil {
		t.Error(err)
	}

	err = net.NetRun()
	if err != nil {
		t.Error(err)
	}

	for i := 0; i < 5; i++ {
		time.Sleep(time.Millisecond)
		err = net.DeviceOutput(dev, []byte{0xff, 0xff, 0x11}, 0x0000, devices.EtherAddrBroadcast)
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

	dev, err := devices.EtherInit("tap0")
	if err != nil {
		t.Error(err)
	}

	err = net.NetRun()
	if err != nil {
		t.Error(err)
	}

	for i := 0; i < 5; i++ {
		time.Sleep(time.Millisecond)
		err = net.DeviceOutput(dev, []byte{0xff, 0xff, 0x11}, 0x0000, devices.EtherAddrBroadcast)
		if err != nil {
			t.Error(err)
		}
	}

	err = net.NetShutdown()
	if err != nil {
		t.Error(err)
	}
}
