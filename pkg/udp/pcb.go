package udp

import (
	"fmt"
	"log"
	"sync"

	"github.com/hedwig100/go-network/pkg/ip"
)

/*
	Protocol Control Block (for socket API)
*/

const (
	pcbStateFree    = 0
	pcbStateOpen    = 1
	pcbStateClosing = 2

	pcbBufSize = 100
)

var (
	mutex sync.Mutex
	pcbs  []*pcb
)

// pcb is protocol control block for UDP
type pcb struct {

	// pcb state
	state int

	// our UDP endpoint
	local Endpoint

	// receive queue
	rxQueue chan buffer
}

// buffer is
type buffer struct {

	// UDP endpoint of the source
	foreign Endpoint

	// data sent to us
	data []byte
}

func pcbSelect(address ip.Addr, port uint16) *pcb {
	for _, p := range pcbs {
		if p.local.Addr == address && p.local.Port == port {
			return p
		}
	}
	return nil
}

func OpenUDP() *pcb {
	pcb := &pcb{
		state: pcbStateOpen,
		local: Endpoint{
			Addr: ip.AddrAny,
		},
		rxQueue: make(chan buffer, pcbBufSize),
	}
	mutex.Lock()
	pcbs = append(pcbs, pcb)
	mutex.Unlock()
	return pcb
}

func Close(pcb *pcb) error {

	index := -1
	mutex.Lock()
	defer mutex.Unlock()
	for i, p := range pcbs {
		if p == pcb {
			index = i
			break
		}
	}
	if index < 0 {
		return fmt.Errorf("pcb not found")
	}

	pcbs = append(pcbs[:index], pcbs[index+1:]...)
	return nil
}

func (pcb *pcb) Bind(local Endpoint) error {

	// check if the same address has not been bound
	mutex.Lock()
	defer mutex.Unlock()
	for _, p := range pcbs {
		if p.local == local {
			return fmt.Errorf("local address(%s) is already binded", local)
		}
	}
	pcb.local = local
	log.Printf("[I] bound address local=%s", local)
	return nil
}

func (pcb *pcb) Send(data []byte, dst Endpoint) error {

	local := pcb.local

	if local.Addr == ip.AddrAny {
		route, err := ip.LookupTable(dst.Addr)
		if err != nil {
			return err
		}
		local.Addr = route.Iface.Unicast
	}

	if local.Port == 0 { // zero value of Port (uint16)
		for p := PortMin; p <= PortMax; p++ {
			if pcbSelect(local.Addr, p) != nil {
				local.Port = p
				log.Printf("[D] registered UDP :address=%s,port=%d", local.Addr, local.Port)
				break
			}
		}
		if local.Port == 0 {
			return fmt.Errorf("there is no port number to assign")
		}
	}

	return TxHandler(local, dst, data)
}

// Listen listens data and write data to 'data'. if 'block' is false, there is no blocking I/O.
// This function returns data size,data,and source UDP endpoint.
func (pcb *pcb) Listen(block bool) (int, []byte, Endpoint) {

	if block {
		buf := <-pcb.rxQueue
		return len(buf.data), buf.data, buf.foreign
	}

	// no blocking
	select {
	case buf := <-pcb.rxQueue:
		return len(buf.data), buf.data, buf.foreign
	default:
		return 0, []byte{}, Endpoint{}
	}
}
