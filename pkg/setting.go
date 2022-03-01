package pkg

import (
	"github.com/hedwig100/go-network/pkg/device"
	"github.com/hedwig100/go-network/pkg/icmp"
	"github.com/hedwig100/go-network/pkg/ip"
	"github.com/hedwig100/go-network/pkg/net"
	"github.com/hedwig100/go-network/pkg/tcp"
	"github.com/hedwig100/go-network/pkg/udp"
)

var done chan struct{} = make(chan struct{})

func NetInit(setup bool) error {

	if setup {
		_ = device.NullInit("null0")
		loop := device.LoopbackInit("loop0")
		ether, err := device.EtherInit("tap0")
		if err != nil {
			return err
		}

		// iface
		iface0, err := ip.NewIface("127.0.0.1", "255.0.0.0")
		if err != nil {
			return err
		}
		ip.IfaceRegister(loop, iface0)

		iface1, err := ip.NewIface("192.0.2.2", "255.255.255.0")
		if err != nil {
			return err
		}
		ip.IfaceRegister(ether, iface1)

		// default gateway
		err = ip.SetDefaultGateway(iface1, "192.0.2.1")
		if err != nil {
			return err
		}
	}

	err := ip.Init(done)
	if err != nil {
		return err
	}

	err = icmp.Init()
	if err != nil {
		return err
	}

	err = udp.Init()
	if err != nil {
		return err
	}

	err = tcp.Init(done)
	if err != nil {
		return err
	}

	return nil
}

func NetRun() {

	// activate the receive handler of the device
	for _, dev := range net.Devices {
		go dev.RxHandler(done)
	}

	// activate the receive handler of the protocol
	for i, proto := range net.Protos {
		go proto.RxHandler(net.ProtoBuffers[i], done)
	}
}

func NetShutdown() (err error) {

	// shutdown all rxHandler
	close(done)

	// close devices
	if err = net.CloseDevices(); err != nil {
		return
	}
	return
}
