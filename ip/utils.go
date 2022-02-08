package ip

import (
	"strconv"
	"strings"
)

var id uint16 = 0

// generateId() generates id for IP header
func generateId() uint16 {
	id++
	return id
}

// Str2IPAddr transforms IP address string to 32bit address
// ex) "127.0.0.1" -> 01111111 00000000 00000000 00000001
func Str2IPAddr(str string) (uint32, error) {
	strs := strings.Split(str, ".")
	var b uint32
	for i, s := range strs {
		n, err := strconv.Atoi(s)
		if err != nil {
			return 0, err
		}
		b |= uint32((n & 0xff) << (24 - 8*i))
	}
	return b, nil
}
