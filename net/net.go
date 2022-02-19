package net

var done chan struct{} = make(chan struct{})

func NetInit(setup bool) error {

	if setup {
		_ = NullInit("null0")
		loop := LoopbackInit("loop0")
		ether, err := EtherInit("tap0")
		if err != nil {
			return err
		}

		// iface
		iface0, err := NewIPIface("127.0.0.1", "255.0.0.0")
		if err != nil {
			return err
		}
		IPIfaceRegister(loop, iface0)

		iface1, err := NewIPIface("192.0.2.2", "255.255.255.0")
		if err != nil {
			return err
		}
		IPIfaceRegister(ether, iface1)

		// default gateway
		err = SetDefaultGateway(iface1, "192.0.2.1")
		if err != nil {
			return err
		}
	}

	err := arpInit(done)
	if err != nil {
		return err
	}

	err = ipInit()
	if err != nil {
		return err
	}

	err = icmpInit()
	if err != nil {
		return err
	}

	err = udpInit()
	if err != nil {
		return err
	}

	err = tcpInit(done)
	if err != nil {
		return err
	}

	return nil
}

func NetRun() {

	// activate the receive handler of the device
	for _, dev := range Devices {
		go dev.rxHandler(done)
	}

	// activate the receive handler of the protocol
	for i, proto := range Protocols {
		go proto.rxHandler(ProtocolBuffers[i], done)
	}
}

func NetShutdown() (err error) {

	// shutdown all rxHandler
	close(done)

	// close devices
	if err = CloseDevices(); err != nil {
		return
	}
	return
}
