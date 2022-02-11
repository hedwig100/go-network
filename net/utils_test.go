package net_test

import (
	"testing"

	"github.com/hedwig100/go-network/net"
)

func TestChecksum(t *testing.T) {
	a := []byte{0x99, 0x01, 0x11, 0x98, 0x00, 0x00}
	chksum := net.CheckSum(a)
	copy(a[4:6], net.Hton16(chksum))
	b := net.CheckSum(a)
	if b != 0 && b != 0xffff {
		t.Errorf("b: %x", b)
	}
}
