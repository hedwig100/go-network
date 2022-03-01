package device_test

import (
	"testing"
	"time"

	"github.com/hedwig100/go-network/net"
	"github.com/hedwig100/go-network/net/device"
)

func TestNull(t *testing.T) {
	var err error

	dev := device.NullInit("null0")

	err = net.NetInit(false)
	if err != nil {
		t.Error(err)
	}

	for i := 0; i < 5; i++ {
		time.Sleep(time.Millisecond)
		err = device.DeviceOutput(dev, []byte{0xff, 0xff, 0x11}, 0x0000, device.EtherAddrBroadcast)
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
	var err error

	dev := device.LoopbackInit("loopback0")

	err = net.NetInit(false)
	if err != nil {
		t.Error(err)
	}

	for i := 0; i < 5; i++ {
		time.Sleep(time.Millisecond)
		err = device.DeviceOutput(dev, []byte{0xff, 0xff, 0x11}, 0x0000, device.EtherAddrBroadcast)
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

	dev, err := device.EtherInit("tap0")
	if err != nil {
		t.Error(err)
	}

	err = net.NetInit(false)
	if err != nil {
		t.Error(err)
	}

	for i := 0; i < 5; i++ {
		time.Sleep(time.Second)
		err = device.DeviceOutput(dev, []byte{0xff, 0xff, 0x11}, 0x0000, device.EtherAddrBroadcast)
		if err != nil {
			t.Error(err)
		}
	}

	err = net.NetShutdown()
	if err != nil {
		t.Error(err)
	}
}
