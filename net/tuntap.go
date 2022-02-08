package net

import (
	"errors"
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

const (
	cloneDevice = "/dev/net/tun"
)

func openTap(name string) (string, *os.File, error) {

	if len(name) >= syscall.IFNAMSIZ {
		return "", nil, errors.New("device name is too long")
	}

	fd, err := unix.Open(cloneDevice, os.O_RDWR, 0600)
	if err != nil {
		return "", nil, err
	}

	name, err = TUNSETIFF(uintptr(fd), name)
	if err != nil {
		return "", nil, err
	}

	// https://github.com/golang/go/issues/30426
	unix.SetNonblock(fd, true)
	file := os.NewFile(uintptr(fd), cloneDevice)

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

func getAddr(name string) (addr [EtherAddrLen]byte, err error) {
	_addr, err := SIOCGIFHWADDR(name)
	if err != nil {
		return
	}
	copy(addr[:], _addr)
	return
}
