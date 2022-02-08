package icmp_test

import (
	"testing"
	"time"

	"github.com/hedwig100/go-network/devices"
	"github.com/hedwig100/go-network/icmp"
	"github.com/hedwig100/go-network/ip"
	"github.com/hedwig100/go-network/net"
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

func TestICMP(t *testing.T) {
	var err error

	// devices
	_, err = devices.NullInit("null0")
	if err != nil {
		t.Error(err)
	}
	loop, err := devices.LoopbackInit("loop0")
	if err != nil {
		t.Error(err)
	}
	ether, err := devices.EtherInit("tap0")
	if err != nil {
		t.Error(err)
	}

	// iface
	iface0, err := ip.NewIPIface(loopbackIPAddr, loopbackNetmask)
	if err != nil {
		t.Error(err)
	}
	ip.IPIfaceRegister(loop, iface0)

	iface1, err := ip.NewIPIface(etherTapIPAddr, etherTapNetmask)
	if err != nil {
		t.Error(err)
	}
	ip.IPIfaceRegister(ether, iface1)

	// default gateway
	err = ip.SetDefaultGateway(iface0, defaultGateway)
	if err != nil {
		return
	}

	err = net.NetRun()
	if err != nil {
		t.Error(err)
	}

	src := iface1.Unicast
	dst := ip.IPAddrBroadcast
	id := uint32(109)
	seq := uint32(0)

	for i := 0; i < 5; i++ {
		time.Sleep(time.Millisecond)
		err = icmp.TxHandler(icmp.ICMPTypeEcho, 0, (id<<16 | seq), testdata, src, dst)
		seq++
		if err != nil {
			t.Error(err)
		}
	}

	err = net.NetShutdown()
	if err != nil {
		t.Error(err)
	}
}
