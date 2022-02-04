package net

var done chan struct{} = make(chan struct{})

func NetRun() (err error) {
	return
}

func NetShutdown() (err error) {

	if err = CloseDevices(); err != nil {
		return
	}
	return
}
