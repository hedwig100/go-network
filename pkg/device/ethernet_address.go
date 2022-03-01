package device

import "fmt"

// EtherAddr implments net.HardwareAddr interface.
// EtherAddr is written in bigEndian.
type EtherAddr [EtherAddrLen]byte

func (a EtherAddr) Addr() []byte {
	return a[:]
}

func (a EtherAddr) String() string {
	return fmt.Sprintf("%x:%x:%x:%x:%x:%x", a[0], a[1], a[2], a[3], a[4], a[5])
}
