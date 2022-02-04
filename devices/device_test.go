package devices

import (
	"testing"
	"time"

	"github.com/hedwig100/go-network/net"
)

func TestNull(t *testing.T) {
	t.Skip()
	var err error

	dev, err := NullInit("null0")
	if err != nil {
		t.Error(err)
	}

	err = net.NetRun()
	if err != nil {
		t.Error(err)
	}

	for i := 0; i < 5; i++ {
		time.Sleep(time.Millisecond)
		err = net.DeviceOutput(dev, []byte{0xff, 0xff, 0x11}, 0x0000, nil)
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

	dev, err := LoopbackInit("loopback0")
	if err != nil {
		t.Error(err)
	}

	err = net.NetRun()
	if err != nil {
		t.Error(err)
	}

	for i := 0; i < 5; i++ {
		time.Sleep(time.Millisecond)
		err = net.DeviceOutput(dev, []byte{0xff, 0xff, 0x11}, 0x0000, nil)
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

	dev, err := EtherInit("tap0")
	if err != nil {
		t.Error(err)
	}

	err = net.NetRun()
	if err != nil {
		t.Error(err)
	}

	for i := 0; i < 5; i++ {
		time.Sleep(time.Millisecond)
		err = net.DeviceOutput(dev, []byte{0xff, 0xff, 0x11}, 0x0000, nil)
		if err != nil {
			t.Error(err)
		}
	}

	err = net.NetShutdown()
	if err != nil {
		t.Error(err)
	}
}
