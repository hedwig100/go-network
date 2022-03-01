package raw

import (
	"errors"
	"os"
	"syscall"
)

const (
	cloneDevice = "/dev/net/tun"
)

// OpenTap open a tap device whose name is 'name' and returns its name,pointer to the file,error
func OpenTap(name string) (string, *os.File, error) {

	if len(name) >= syscall.IFNAMSIZ {
		return "", nil, errors.New("device name is too long")
	}

	file, err := os.OpenFile(cloneDevice, os.O_RDWR, 0600)
	if err != nil {
		return "", nil, err
	}

	name, err = TUNSETIFF(file.Fd(), name, syscall.IFF_TAP|syscall.IFF_NO_PI)
	if err != nil {
		return "", nil, err
	}

	flags, err := SIOCGIFFLAGS(name)
	if err != nil {
		return "", nil, err
	}

	flags |= syscall.IFF_UP | syscall.IFF_RUNNING
	err = SIOCSIFFLAGS(name, flags)
	if err != nil {
		return "", nil, err
	}

	return name, file, nil
}

// GetAddr returns ethernet address of the tap device whose name is 'name'
func GetAddr(name string) ([]byte, error) {
	addr, err := SIOCGIFHWADDR(name)
	if err != nil {
		return nil, err
	}
	return addr, nil
}
