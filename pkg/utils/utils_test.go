package utils_test

import (
	"testing"

	"github.com/hedwig100/go-network/pkg/utils"
)

func TestChecksum(t *testing.T) {
	a := []byte{0x99, 0x01, 0x11, 0x98, 0x00, 0x00}
	chksum := utils.CheckSum(a, 0)
	copy(a[4:6], utils.Hton16(chksum))
	b := utils.CheckSum(a, 0)
	if b != 0 && b != 0xffff {
		t.Errorf("b: %x", b)
	}
}
