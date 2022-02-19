package net

var done chan struct{} = make(chan struct{})

func NetInit() error {

	err := ArpInit(done)
	if err != nil {
		return err
	}

	err = IPInit()
	if err != nil {
		return err
	}

	err = ICMPInit()
	if err != nil {
		return err
	}

	err = UDPInit()
	if err != nil {
		return err
	}

	err = TCPInit(done)
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
