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
		iface0, err := ip.NewIPIface("127.0.0.1", "255.0.0.0")
		if err != nil {
			return err
		}
		ip.IPIfaceRegister(loop, iface0)

		iface1, err := ip.NewIPIface("192.0.2.2", "255.255.255.0")
		if err != nil {
			return err
		}
		ip.IPIfaceRegister(ether, iface1)

		// default gateway
		err = ip.SetDefaultGateway(iface1, "192.0.2.1")
		if err != nil {
			return err
		}
	}

	err := ip.ArpInit(done)
	if err != nil {
		return err
	}

	err = ip.IPInit()
	if err != nil {
		return err
	}

	err = icmp.ICMPInit()
	if err != nil {
		return err
	}

	err = udp.UDPInit()
	if err != nil {
		return err
	}

	err = tcp.TCPInit(done)
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
	for i, proto := range net.Protocols {
		go proto.RxHandler(net.ProtocolBuffers[i], done)
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