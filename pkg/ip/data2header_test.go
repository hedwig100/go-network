package ip

import (
	"log"
	"testing"
)

func compareByte(a []byte, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func Test2IP(t *testing.T) {
	src, _ := Str2Addr("127.0.0.1")
	dst, _ := Str2Addr("8.8.8.8")

	org_hdr := Header{
		Vhl:       V4<<4 | HeaderSizeMin>>2,
		Tos:       0xff,
		Tol:       HeaderSizeMin + 3,
		Id:        1,
		Flags:     0,
		Ttl:       64,
		ProtoType: ProtoICMP,
		Checksum:  0,
		Src:       Addr(src),
		Dst:       Addr(dst),
	}
	org_payload := []byte{0x92, 0x12, 0x29}

	data, err := header2data(&org_hdr, org_payload)
	if err != nil {
		t.Error(err)
	}

	new_hdr, new_payload, err := data2header(data)
	if err != nil {
		t.Error(err)
	}

	log.Printf("%s\n", org_hdr)
	log.Println(org_payload)
	log.Println(data)
	log.Printf("%s\n", new_hdr)
	log.Println(new_payload)

	if org_hdr != new_hdr {
		t.Error("IPv4 header transform not succeeded")
	}
	if !compareByte(org_payload, new_payload) {
		t.Error("IPv4 payload transforrm not succeeded")
	}
}
