package icmp

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
func Test2ICMP(t *testing.T) {
	org_hdr := ICMPHeader{
		Typ:      ICMPTypeDestUnreach,
		Code:     ICMPCodeNetUnreach,
		Checksum: 0,
		Values:   19,
	}
	org_payload := []byte{90, 21, 143, 134}

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
		t.Error("ICMP header transform not succeeded")
	}
	if !compareByte(org_payload, new_payload) {
		t.Error("ICMP payload transforrm not succeeded")
	}
}
