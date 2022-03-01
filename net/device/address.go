package device

// HardwareAddress is the abstraction of the hardware address such as MAC address
type HardwareAddress interface {

	// Address return address
	Address() []byte

	// String is for printing address
	String() string
}
