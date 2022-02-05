package utils

import "fmt"

func Hton16(v uint16) (b []byte) {
	b = make([]byte, 2)
	b[0] = byte(v >> 8)
	b[1] = byte(v)
	return
}

// bを16bitごとに1の補数和を取り, 最後にその1の補数を取る
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
