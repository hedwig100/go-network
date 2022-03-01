package utils

func Max(a uint16, b uint16) uint16 {
	if a > b {
		return a
	}
	return b
}

// Hton16 transforms 16bit littleEndian number to 16bit bigEndian
func Hton16(v uint16) []byte {
	b := make([]byte, 2)
	b[0] = byte(v >> 8)
	b[1] = byte(v)
	return b
}

// Ntoh16 transforms 16bit bigEndian number to 16bit littleEndian
func Ntoh16(b []byte) uint16 {
	return uint16(b[1])<<8 | uint16(b[0])
}

// Hton32 transforms 32bit littleEndian number to 32bit bigEndian
func Hton32(v uint32) []byte {
	b := make([]byte, 4)
	b[0] = byte(v >> 24)
	b[1] = byte(v >> 16)
	b[2] = byte(v >> 8)
	b[3] = byte(v)
	return b
}

// Ntoh32 transforms 32bit bigEndian number to 32bit littleEndian
func Ntoh32(v uint32) uint32 {
	return (v&0xff)<<24 | (v&0xff00)<<8 | (v&0xff0000)>>8 | (v >> 24)
}

// Checksum calculates the 16-bit 1's complement sum of 1's complement.
// https://datatracker.ietf.org/doc/html/rfc1071
func CheckSum(b []byte, base uint32) uint16 {
	if len(b)%2 == 1 {
		b = append(b, 0)
	}

	ret := base
	for i := 0; i < len(b); i += 2 {
		ret += (uint32(b[i])<<8 | uint32(b[i+1]))
	}

	for (ret >> 16) > 0 {
		ret = (ret & 0xffff) + uint32(ret>>16)
	}
	return ^uint16(ret)
}
