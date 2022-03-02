package udp

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/hedwig100/go-network/pkg/ip"
)

const (
	PortMin uint16 = 49152
	PortMax uint16 = 65535
)

// Endpoint is IP address and port number combination
type Endpoint struct {

	// IP address
	Addr ip.Addr

	// port number
	Port uint16
}

func (e Endpoint) String() string {
	return fmt.Sprintf("%s:%d", e.Addr, e.Port)
}

// Str2Endpoint encodes str to Endpoint
// ex) str="8.8.8.8:80"
func Str2Endpoint(str string) (Endpoint, error) {
	tmp := strings.Split(str, ":")
	if len(tmp) != 2 {
		return Endpoint{}, fmt.Errorf("str is not correect")
	}
	addr, err := ip.Str2Addr(tmp[0])
	if err != nil {
		return Endpoint{}, err
	}
	port, err := strconv.Atoi(tmp[1])
	if err != nil {
		return Endpoint{}, err
	}
	return Endpoint{
		Addr: ip.Addr(addr),
		Port: uint16(port),
	}, nil
}
