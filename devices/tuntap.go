package devices

import (
	"errors"
	"os"
	"syscall"
)

const (
	cloneDevice = "dev/net/tun"
)

func openTap(name string) (string, *os.File, error) {

	if len(name) >= syscall.IFNAMSIZ {
		return "", nil, errors.New("device name is too long")
	}

	fd, err := os.OpenFile(cloneDevice, os.O_RDWR, 0600)
	if err != nil {
		return "", nil, err
	}

	name, err = TUNSETIFF(fd.Fd(), name)
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

	return name, fd, nil
}

func getAddr(name string) (addr [EtherAddrLen]byte, err error) {
	_addr, err := SIOCGIFHWADDR(name)
	if err != nil {
		return
	}
	copy(addr[:], _addr)
	return
}
