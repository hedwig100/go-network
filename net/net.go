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

	err = TCPInit()
	if err != nil {
		return err
	}

	return nil
}

func NetRun() {

	// activate the receive handler of the device
	for _, dev := range Devices {
		go dev.RxHandler(done)
	}

	// activate the receive handler of the protocol
	for i, proto := range Protocols {
		go proto.RxHandler(ProtocolBuffers[i], done)
	}
}

func NetShutdown() (err error) {

	// shutdown all RxHandler
	close(done)

	// close devices
	if err = CloseDevices(); err != nil {
		return
	}
	return
}
