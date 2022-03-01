package tcp

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/hedwig100/go-network/pkg/ip"
)

/*
	Retransmition Queue entry
*/
const (
	maxRetxCount uint8 = 3

	triggerNo      uint8 = 0
	triggerOpen    uint8 = 1
	triggerClose   uint8 = 2
	triggerReceive uint8 = 3
	triggerSend    uint8 = 4
)

type retxEntry struct {
	data        []byte
	seq         uint32
	flag        ControlFlag
	first       time.Time
	last        time.Time
	retxCount   uint8
	errCh       chan error
	triggerType uint8
}

type rcvCmd struct {
	n     *int
	data  []byte
	errCh chan error
}

/*
	TCP Protocol Control Block (Transmission Control Block)
*/

const (
	PCBStateListen      PCBState = 0
	PCBStateSYNSent     PCBState = 1
	PCBStateSYNReceived PCBState = 2
	PCBStateEstablished PCBState = 3
	PCBStateFINWait1    PCBState = 4
	PCBStateFINWait2    PCBState = 5
	PCBStateCloseWait   PCBState = 6
	PCBStateClosing     PCBState = 7
	PCBStateLastACK     PCBState = 8
	PCBStateTimeWait    PCBState = 9
	PCBStateClosed      PCBState = 10
)

type PCBState uint32

func (s PCBState) String() string {
	switch s {
	case PCBStateListen:
		return "LISTEN"
	case PCBStateSYNSent:
		return "SYN-SENT"
	case PCBStateSYNReceived:
		return "SYN-RECEIVED"
	case PCBStateEstablished:
		return "ESTABLISHED"
	case PCBStateFINWait1:
		return "FIN-WAIT-1"
	case PCBStateFINWait2:
		return "FIN-WAIT-2"
	case PCBStateCloseWait:
		return "CLOSE-WAIT"
	case PCBStateClosing:
		return "CLOSING"
	case PCBStateLastACK:
		return "LAST-ACK"
	case PCBStateTimeWait:
		return "TIME-WAIT"
	case PCBStateClosed:
		return "CLOSED"
	default:
		return "UNKNOWN"
	}
}

func createISS() uint32 {
	return rand.Uint32()
}

type snd struct {
	una uint32
	nxt uint32
	wnd uint16
	up  uint16
	wl1 uint32
	wl2 uint32
}

type rcv struct {
	nxt uint32
	wnd uint16
	up  uint16
}

const (
	bufferSize = math.MaxUint16
)

var (
	mutex sync.Mutex
	pcbs  []*pcb
)

type pcb struct {
	state   PCBState
	local   Endpoint
	foreign Endpoint

	snd
	iss uint32
	rcv
	irs uint32

	mss uint16

	// queue
	rxQueue   [bufferSize]byte
	rxLen     uint16
	retxQueue []retxEntry
	rcvCmd    rcvCmd

	timeout    time.Duration
	lastTxTime time.Time
}

func (pcb *pcb) transition(state PCBState) {
	log.Printf("[I] local=%s, %s => %s", pcb.local, pcb.state, state)
	pcb.state = state
}

func (pcb *pcb) queueAdd(seq uint32, flag ControlFlag, data []byte, trigger uint8, errCh chan error) {
	pcb.retxQueue = append(pcb.retxQueue, retxEntry{
		data:        data,
		seq:         seq,
		flag:        flag,
		first:       time.Now(),
		last:        time.Now(),
		triggerType: trigger,
		errCh:       errCh,
	})
}

func removeRetx(data []retxEntry, indexs []int) []retxEntry {
	if len(indexs) == 0 {
		return data
	}
	ret := data[:indexs[0]]
	for i := 1; i < len(indexs); i++ {
		ret = append(ret, data[indexs[i-1]+1:indexs[i]]...)
	}
	ret = append(ret, data[indexs[len(indexs)-1]+1:]...)
	return ret
}

func (pcb *pcb) queueAck() {
	var deleteIndex []int
	for i, entry := range pcb.retxQueue {
		if entry.seq < pcb.una {
			deleteIndex = append(deleteIndex, i)
			if entry.errCh != nil {
				entry.errCh <- nil
			}
			calculateRTO(time.Since(entry.last))
		}
	}
	pcb.retxQueue = removeRetx(pcb.retxQueue, deleteIndex)
}

func (pcb *pcb) queueFlush(msg string) {
	err := fmt.Errorf(msg)
	for _, entry := range pcb.retxQueue {
		if entry.errCh != nil {
			entry.errCh <- err
		}
	}
	pcb.retxQueue = nil
}

func (pcb *pcb) signalCmd(trigger uint8) {

	switch trigger {
	case triggerOpen, triggerClose:
		var deleteIndex []int
		for i, entry := range pcb.retxQueue {
			if entry.triggerType == trigger {
				entry.errCh <- nil
				deleteIndex = append(deleteIndex, i)
			}
		}
		pcb.retxQueue = removeRetx(pcb.retxQueue, deleteIndex)
	case triggerReceive:
		if pcb.rcvCmd.errCh != nil && pcb.rxLen > 0 {
			dlen := copy(pcb.rcvCmd.data, pcb.rxQueue[:pcb.rxLen])
			*pcb.rcvCmd.n = dlen
			pcb.rcvCmd.errCh <- nil
			if dlen == int(pcb.rxLen) {
				pcb.rxLen = 0
				pcb.rcv.wnd = bufferSize
			} else {
				copy(pcb.rxQueue[:], pcb.rxQueue[pcb.rxLen-uint16(dlen):])
				pcb.rxLen -= uint16(dlen)
				pcb.rcv.wnd += uint16(dlen)
			}
			pcb.rcvCmd = rcvCmd{}
		}
	default:
	}
}

func (pcb *pcb) signalErr(msg string) {
	pcb.queueFlush(msg)
	err := fmt.Errorf(msg)
	if pcb.rcvCmd.errCh != nil {
		pcb.rcvCmd.errCh <- err
	}
	pcb.rcvCmd = rcvCmd{}
}

// Newpcb returns *TCBpcb if there is no *pcb whose address is not the same as local
func Newpcb(local Endpoint) (*pcb, error) {
	// check if the same local address has not been used
	mutex.Lock()
	defer mutex.Unlock()
	for _, t := range pcbs {
		if t.local == local {
			return nil, fmt.Errorf("the same local address(%s) is already used", local)
		}
	}

	pcb := &pcb{
		state: PCBStateClosed,
		local: local,
	}
	pcbs = append(pcbs, pcb)
	return pcb, nil
}

func Deletepcb(pcb *pcb) error {
	mutex.Lock()
	defer mutex.Unlock()
	for i, t := range pcbs {
		if t == pcb {
			pcbs = append(pcbs[:i], pcbs[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("pcb not found, and cannot be deleted")
}

func (pcb *pcb) Open(errCh chan error, foreign Endpoint, isActive bool, timeout time.Duration) {
	mutex.Lock()
	defer mutex.Unlock()

	switch pcb.state {
	case PCBStateClosed:
		// passive open
		if !isActive {
			log.Printf("[D] passive open: local=%s,waiting for connection...", pcb.local)
			pcb.timeout = timeout
			pcb.transition(PCBStateListen)
			errCh <- nil
			return
		}
		// active open
		if foreign.Addr == ip.AddrAny {
			errCh <- fmt.Errorf("foreign socket unspecified")
			return
		}

		pcb.timeout = timeout
		pcb.foreign = foreign

		iss := createISS()
		pcb.iss = iss
		pcb.snd.una = iss
		pcb.snd.nxt = iss + 1

		var err error
		for i := 0; i < 3; i++ { // try to send SYN at most three time ( because of ARP cache specification of this package).
			if err = TxHelperTCP(pcb, SYN, []byte{}, triggerOpen, errCh); err != nil {
				log.Printf("[E] TCP OPEN call error %s", err.Error())
				time.Sleep(20 * time.Millisecond)
			} else {
				pcb.transition(PCBStateSYNSent)
				log.Printf("[D] active open: local=%s,foreign=%s,connecting...", pcb.local, pcb.foreign)
				return
			}
		}
		errCh <- err

	case PCBStateListen:
		// passive open
		if !isActive {
			log.Printf("[D] passive open: local=%s,waiting for connection...", pcb.local)
			pcb.timeout = timeout
			errCh <- nil
			return
		}
		// active open
		if foreign.Addr == ip.AddrAny {
			errCh <- fmt.Errorf("foreign socket unspecified")
			return
		}

		pcb.timeout = timeout
		pcb.foreign = foreign

		iss := createISS()
		pcb.iss = iss
		pcb.snd.una = iss
		pcb.snd.nxt = iss + 1

		var err error
		for i := 0; i < 3; i++ { // try to send SYN at most three time ( because of ARP cache specification of this package).
			if err = TxHelperTCP(pcb, SYN, []byte{}, triggerOpen, errCh); err != nil {
				log.Printf("[E] TCP OPEN call error %s", err.Error())
				time.Sleep(20 * time.Millisecond)
			} else {
				pcb.transition(PCBStateSYNSent)
				log.Printf("[D] active open: local=%s,foreign=%s,connecting...", pcb.local, pcb.foreign)
				return
			}
		}
		errCh <- err

	default:
		errCh <- fmt.Errorf("connection already exists")
	}
}

func (pcb *pcb) Send(errCh chan error, data []byte) {
	mutex.Lock()
	defer mutex.Unlock()

	switch pcb.state {
	case PCBStateClosed:
		errCh <- fmt.Errorf("connection does not exist")
	case PCBStateListen:
		errCh <- fmt.Errorf("connection does not exist")
	case PCBStateSYNSent, PCBStateSYNReceived:
		errCh <- fmt.Errorf("connection does not exist")
	case PCBStateEstablished, PCBStateCloseWait:
		if len(data) > int(pcb.snd.wnd) {
			errCh <- fmt.Errorf("insufficient resources")
		}

		var err error
		for i := 0; i < 3; i++ { // try to send data at most three time ( because of ARP cache specification of this package).
			if err = TxHelperTCP(pcb, ACK, data, triggerSend, errCh); err != nil {
				log.Printf("[E] TCP SEND call error %s", err.Error())
				time.Sleep(20 * time.Millisecond)
			} else {
				pcb.snd.nxt += uint32(len(data))
				return
			}
		}
		errCh <- err

	default:
		errCh <- fmt.Errorf("connection closing")
	}
}

func (pcb *pcb) Receive(errCh chan error, buf []byte, n *int) {
	mutex.Lock()
	defer mutex.Unlock()

	switch pcb.state {
	case PCBStateClosed:
		errCh <- fmt.Errorf("connection does not exist")
	case PCBStateListen, PCBStateSYNSent, PCBStateSYNReceived:
		errCh <- fmt.Errorf("connection does not exist")
	case PCBStateEstablished, PCBStateFINWait1, PCBStateFINWait2:
		// If insufficient incoming segments are queued to satisfy the
		// request, queue the request.
		if pcb.rcvCmd.errCh != nil {
			errCh <- fmt.Errorf("RECEIVE was already called and data haven't come yet")
			return
		}
		pcb.rcvCmd = rcvCmd{
			n:     n,
			data:  buf,
			errCh: errCh,
		}
		pcb.signalCmd(triggerReceive)
	case PCBStateCloseWait:
		if pcb.rcvCmd.errCh != nil {
			errCh <- fmt.Errorf("RECEIVE was already called and data haven't come yet")
			return
		}
		pcb.rcvCmd = rcvCmd{
			n:     n,
			data:  buf,
			errCh: errCh,
		}
		pcb.signalCmd(triggerReceive)

		// no remaining data
		if pcb.rcvCmd.errCh != nil {
			pcb.rcvCmd = rcvCmd{}
			errCh <- fmt.Errorf("no remaning data")
		}
	default:
		errCh <- fmt.Errorf("connection closing")
	}
}

func (pcb *pcb) Close(errCh chan error) {
	mutex.Lock()
	defer mutex.Unlock()

	switch pcb.state {
	case PCBStateClosed:
		errCh <- fmt.Errorf("connection does not exist")
	case PCBStateListen:
		// Any outstanding RECEIVEs are returned with "error:  closing" responses.
		pcb.signalErr("closing")
		pcb.transition(PCBStateClosed)
		errCh <- nil
	case PCBStateSYNSent:
		// return "error:  closing" responses to any queued SENDs, or RECEIVEs.
		pcb.signalErr("closing")
		pcb.transition(PCBStateClosed)
		errCh <- nil
	case PCBStateSYNReceived:
		// If no SENDs have been issued and there is no pending data to send,
		// then form a FIN segment and send it, and enter FIN-WAIT-1 state;
		// otherwise queue for processing after entering ESTABLISHED state.
		var err error
		for i := 0; i < 3; i++ { // try to send FIN at most three time ( because of ARP cache specification of this package).
			if err = TxHelperTCP(pcb, ACK|FIN, []byte{}, triggerClose, errCh); err != nil {
				log.Printf("[E] TCP CLOSE call error %s", err.Error())
				time.Sleep(20 * time.Millisecond)
			} else {
				pcb.transition(PCBStateFINWait1)
				pcb.snd.nxt++
				log.Printf("[D] active close: local=%s,foreign=%s,closing...", pcb.local, pcb.foreign)
				return
			}
		}
		errCh <- err

	case PCBStateEstablished:
		// Queue this until all preceding SENDs have been segmentized, then
		// form a FIN segment and send it.  In any case, enter FIN-WAIT-1
		// state.
		var err error
		for i := 0; i < 3; i++ { // try to send FIN at most three time ( because of ARP cache specification of this package).
			if err = TxHelperTCP(pcb, ACK|FIN, []byte{}, triggerClose, errCh); err != nil {
				log.Printf("[E] TCP CLOSE call error %s", err.Error())
				time.Sleep(20 * time.Millisecond)
			} else {
				pcb.transition(PCBStateFINWait1)
				pcb.snd.nxt++
				log.Printf("[D] active close: local=%s,foreign=%s,closing...", pcb.local, pcb.foreign)
				return
			}
		}
		errCh <- err

	case PCBStateFINWait1, PCBStateFINWait2:
		errCh <- fmt.Errorf("connection closing")
	case PCBStateCloseWait:
		// Queue this request until all preceding SENDs have been
		// segmentized; then send a FIN segment, enter CLOSING state.
		var err error
		for i := 0; i < 3; i++ { // try to send FIN at most three time ( because of ARP cache specification of this package).
			if err = TxHelperTCP(pcb, ACK|FIN, []byte{}, triggerClose, errCh); err != nil {
				log.Printf("[E] TCP CLOSE call error %s", err.Error())
				time.Sleep(20 * time.Millisecond)
			} else {
				pcb.transition(PCBStateLastACK)
				pcb.snd.nxt++
				return
			}
		}
		errCh <- err

	default:
		errCh <- fmt.Errorf("connection closing")
	}
}

func (pcb *pcb) Abort() error {
	mutex.Lock()
	defer mutex.Unlock()

	switch pcb.state {
	case PCBStateClosed:
		return fmt.Errorf("connection does not exist")
	case PCBStateListen:
		// Any outstanding RECEIVEs should be returned with "error:
		// connection reset" responses
		pcb.signalErr("connection reset")
		pcb.transition(PCBStateClosed)
		return nil
	case PCBStateSYNSent:
		// All queued SENDs and RECEIVEs should be given "connection reset" notification,
		// RECEIVE
		pcb.signalErr("connection reset")
		pcb.transition(PCBStateClosed)
		return nil
	case PCBStateSYNReceived, PCBStateEstablished, PCBStateFINWait1, PCBStateFINWait2, PCBStateCloseWait:
		// All queued SENDs and RECEIVEs should be given "connection reset"
		// notification; all segments queued for transmission (except for the
		// RST formed above) or retransmission should be flushed
		pcb.signalErr("connection reset")
		pcb.transition(PCBStateClosed)
		return TxHandlerTCP(pcb.local, pcb.foreign, []byte{}, pcb.snd.nxt, 0, RST, 0, 0)
	default:
		pcb.transition(PCBStateClosed)
		return nil
	}
}

func (pcb *pcb) Status() PCBState {
	return pcb.state
}
