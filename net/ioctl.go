package net

import (
	"bytes"
	"syscall"
	"unsafe"
)

type ifreq struct {
	name  [syscall.IFNAMSIZ]byte
	flags uint16
	pad   [22]byte // nonnecessary area
}

// デバイスの名前を取得する
// get the device name
// char *dev should be the name of the device with a format string (e.g.
// "tun%d"), but (as far as I can see) this can be any valid network device name.
// Note that the character pointer becomes overwritten with the real device name
// (e.g. "tun0")
func TUNSETIFF(fd uintptr, name string) (string, error) {
	var ifr ifreq
	ifr.flags = syscall.IFF_TAP | syscall.IFF_NO_PI // tap and no-packet-information
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, syscall.TUNSETIFF, uintptr(unsafe.Pointer(&ifr))); errno != 0 {
		return "", errno
	}
	return string(ifr.name[:bytes.IndexByte(ifr.name[:], 0)]), nil
}

// デバイスの active フラグワードを取得する
// get active flag word of the device (whose name is "name")
func SIOCGIFFLAGS(name string) (uint16, error) {

	// ソケットを開く: open the socket
	soc, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_DGRAM, 0)
	if err != nil {
		return 0, err
	}
	defer syscall.Close(soc)

	// リクエストにデバイスの名前を設定: set the name of the device in the ifruest
	var ifr ifreq
	copy(ifr.name[:syscall.IFNAMSIZ-1], name)

	// システムコールを呼んでリクエストのフラグを受け取る: call the system call and receive the flags of the request
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(soc), syscall.SIOCGIFFLAGS, uintptr(unsafe.Pointer(&ifr))); errno != 0 {
		return 0, errno
	}

	return ifr.flags, nil
}

// デバイスの active フラグワードを設定する
// set active flag word of the device (whose name is "name")
func SIOCSIFFLAGS(name string, flag uint16) error {

	// ソケットを開く: open the socket
	soc, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_DGRAM, 0)
	if err != nil {
		return err
	}
	defer syscall.Close(soc)

	// リクエストにデバイスの名前,フラグを設定: set the name of the device and flags in the request
	var ifr ifreq
	copy(ifr.name[:syscall.IFNAMSIZ-1], name)
	ifr.flags = flag

	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(soc), syscall.SIOCSIFFLAGS, uintptr(unsafe.Pointer(&ifr))); errno != 0 {
		return errno
	}

	return nil
}

type sockaddr struct {
	family uint16
	addr   [14]byte
}

// デバイスのハードウェアアドレスを取得する
// get hardware address of the device
func SIOCGIFHWADDR(name string) ([]byte, error) {
	soc, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_DGRAM, 0)
	if err != nil {
		return nil, err
	}
	defer syscall.Close(soc)

	ifr := struct {
		name [syscall.IFNAMSIZ]byte
		addr sockaddr
		_pad [8]byte
	}{}
	copy(ifr.name[:syscall.IFNAMSIZ-1], name)

	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(soc), syscall.SIOCGIFHWADDR, uintptr(unsafe.Pointer(&ifr))); errno != 0 {
		return nil, errno
	}
	return ifr.addr.addr[:], nil
}
