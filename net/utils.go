package net

import (
	"fmt"
	"strconv"
	"strings"
)

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

func Hton16(v uint16) []byte {
	b := make([]byte, 2)
	b[0] = byte(v >> 8)
	b[1] = byte(v)
	return b
}

func Ntoh16(b []byte) uint16 {
	return uint16(b[1])<<8 | uint16(b[0])
}

func NtoH32(v uint32) uint32 {
	return (v&0xff)<<24 | (v&0xff00)<<8 | (v&0xff0000)>>8 | (v >> 24)
}

// the 16-bit 1's complement sum of 1's complement
// https://datatracker.ietf.org/doc/html/rfc1071
func CheckSum(b []byte) uint16 {
	if len(b)%2 == 1 {
		b = append(b, 0)
	}

	var ret uint32
	fmt.Printf("%x\n", ret)
	for i := 0; i < len(b); i += 2 {
		ret += 0xffff ^ (uint32(b[i])<<8 | uint32(b[i+1]))
	}

	for (ret >> 16) > 0 {
		ret = (ret & 0xffff) + uint32(ret>>16)
	}
	return ^uint16(ret)
}