package net

func NetRun() (err error) {

	// if err = OpenDevices(); err != nil {
	// 	return
	// }
	return
}

func NetShutdown() (err error) {

	if err = CloseDevices(); err != nil {
		return
	}
	return
}
