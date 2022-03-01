package net

// HardwareAddr is the abstraction of the hardware address such as MAC address
type HardwareAddr interface {

	// Addr return address
	Addr() []byte

	// String is for printing address
	String() string
}
