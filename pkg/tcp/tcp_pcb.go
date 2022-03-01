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
	TCPpcbStateListen      TCPpcbState = 0
	TCPpcbStateSYNSent     TCPpcbState = 1
	TCPpcbStateSYNReceived TCPpcbState = 2
	TCPpcbStateEstablished TCPpcbState = 3
	TCPpcbStateFINWait1    TCPpcbState = 4
	TCPpcbStateFINWait2    TCPpcbState = 5
	TCPpcbStateCloseWait   TCPpcbState = 6
	TCPpcbStateClosing     TCPpcbState = 7
	TCPpcbStateLastACK     TCPpcbState = 8
	TCPpcbStateTimeWait    TCPpcbState = 9
	TCPpcbStateClosed      TCPpcbState = 10
)

type TCPpcbState uint32

func (s TCPpcbState) String() string {
	switch s {
	case TCPpcbStateListen:
		return "LISTEN"
	case TCPpcbStateSYNSent:
		return "SYN-SENT"
	case TCPpcbStateSYNReceived:
		return "SYN-RECEIVED"
	case TCPpcbStateEstablished:
		return "ESTABLISHED"
	case TCPpcbStateFINWait1:
		return "FIN-WAIT-1"
	case TCPpcbStateFINWait2:
		return "FIN-WAIT-2"
	case TCPpcbStateCloseWait:
		return "CLOSE-WAIT"
	case TCPpcbStateClosing:
		return "CLOSING"
	case TCPpcbStateLastACK:
		return "LAST-ACK"
	case TCPpcbStateTimeWait:
		return "TIME-WAIT"
	case TCPpcbStateClosed:
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
	tcpMutex sync.Mutex
	tcbs     []*TCPpcb
)

type TCPpcb struct {
	state   TCPpcbState
	local   TCPEndpoint
	foreign TCPEndpoint

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

func (tcb *TCPpcb) transition(state TCPpcbState) {
	log.Printf("[I] local=%s, %s => %s", tcb.local, tcb.state, state)
	tcb.state = state
}

func (tcb *TCPpcb) queueAdd(seq uint32, flag ControlFlag, data []byte, trigger uint8, errCh chan error) {
	tcb.retxQueue = append(tcb.retxQueue, retxEntry{
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

func (tcb *TCPpcb) queueAck() {
	var deleteIndex []int
	for i, entry := range tcb.retxQueue {
		if entry.seq < tcb.una {
			deleteIndex = append(deleteIndex, i)
			if entry.errCh != nil {
				entry.errCh <- nil
			}
			calculateRTO(time.Since(entry.last))
		}
	}
	tcb.retxQueue = removeRetx(tcb.retxQueue, deleteIndex)
}

func (tcb *TCPpcb) queueFlush(msg string) {
	err := fmt.Errorf(msg)
	for _, entry := range tcb.retxQueue {
		if entry.errCh != nil {
			entry.errCh <- err
		}
	}
	tcb.retxQueue = nil
}

func (tcb *TCPpcb) signalCmd(trigger uint8) {

	switch trigger {
	case triggerOpen, triggerClose:
		var deleteIndex []int
		for i, entry := range tcb.retxQueue {
			if entry.triggerType == trigger {
				entry.errCh <- nil
				deleteIndex = append(deleteIndex, i)
			}
		}
		tcb.retxQueue = removeRetx(tcb.retxQueue, deleteIndex)
	case triggerReceive:
		if tcb.rcvCmd.errCh != nil && tcb.rxLen > 0 {
			dlen := copy(tcb.rcvCmd.data, tcb.rxQueue[:tcb.rxLen])
			*tcb.rcvCmd.n = dlen
			tcb.rcvCmd.errCh <- nil
			if dlen == int(tcb.rxLen) {
				tcb.rxLen = 0
				tcb.rcv.wnd = bufferSize
			} else {
				copy(tcb.rxQueue[:], tcb.rxQueue[tcb.rxLen-uint16(dlen):])
				tcb.rxLen -= uint16(dlen)
				tcb.rcv.wnd += uint16(dlen)
			}
			tcb.rcvCmd = rcvCmd{}
		}
	default:
	}
}

func (tcb *TCPpcb) signalErr(msg string) {
	tcb.queueFlush(msg)
	err := fmt.Errorf(msg)
	if tcb.rcvCmd.errCh != nil {
		tcb.rcvCmd.errCh <- err
	}
	tcb.rcvCmd = rcvCmd{}
}

// NewTCPpcb returns *TCBpcb if there is no *TCPpcb whose address is not the same as local
func NewTCPpcb(local TCPEndpoint) (*TCPpcb, error) {
	// check if the same local address has not been used
	tcpMutex.Lock()
	defer tcpMutex.Unlock()
	for _, t := range tcbs {
		if t.local == local {
			return nil, fmt.Errorf("the same local address(%s) is already used", local)
		}
	}

	tcb := &TCPpcb{
		state: TCPpcbStateClosed,
		local: local,
	}
	tcbs = append(tcbs, tcb)
	return tcb, nil
}

func DeleteTCPpcb(tcb *TCPpcb) error {
	tcpMutex.Lock()
	defer tcpMutex.Unlock()
	for i, t := range tcbs {
		if t == tcb {
			tcbs = append(tcbs[:i], tcbs[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("tcb not found, and cannot be deleted")
}

func (tcb *TCPpcb) Open(errCh chan error, foreign TCPEndpoint, isActive bool, timeout time.Duration) {
	tcpMutex.Lock()
	defer tcpMutex.Unlock()

	switch tcb.state {
	case TCPpcbStateClosed:
		// passive open
		if !isActive {
			log.Printf("[D] passive open: local=%s,waiting for connection...", tcb.local)
			tcb.timeout = timeout
			tcb.transition(TCPpcbStateListen)
			errCh <- nil
			return
		}
		// active open
		if foreign.Address == ip.AddrAny {
			errCh <- fmt.Errorf("foreign socket unspecified")
			return
		}

		tcb.timeout = timeout
		tcb.foreign = foreign

		iss := createISS()
		tcb.iss = iss
		tcb.snd.una = iss
		tcb.snd.nxt = iss + 1

		var err error
		for i := 0; i < 3; i++ { // try to send SYN at most three time ( because of ARP cache specification of this package).
			if err = TxHelperTCP(tcb, SYN, []byte{}, triggerOpen, errCh); err != nil {
				log.Printf("[E] TCP OPEN call error %s", err.Error())
				time.Sleep(20 * time.Millisecond)
			} else {
				tcb.transition(TCPpcbStateSYNSent)
				log.Printf("[D] active open: local=%s,foreign=%s,connecting...", tcb.local, tcb.foreign)
				return
			}
		}
		errCh <- err

	case TCPpcbStateListen:
		// passive open
		if !isActive {
			log.Printf("[D] passive open: local=%s,waiting for connection...", tcb.local)
			tcb.timeout = timeout
			errCh <- nil
			return
		}
		// active open
		if foreign.Address == ip.AddrAny {
			errCh <- fmt.Errorf("foreign socket unspecified")
			return
		}

		tcb.timeout = timeout
		tcb.foreign = foreign

		iss := createISS()
		tcb.iss = iss
		tcb.snd.una = iss
		tcb.snd.nxt = iss + 1

		var err error
		for i := 0; i < 3; i++ { // try to send SYN at most three time ( because of ARP cache specification of this package).
			if err = TxHelperTCP(tcb, SYN, []byte{}, triggerOpen, errCh); err != nil {
				log.Printf("[E] TCP OPEN call error %s", err.Error())
				time.Sleep(20 * time.Millisecond)
			} else {
				tcb.transition(TCPpcbStateSYNSent)
				log.Printf("[D] active open: local=%s,foreign=%s,connecting...", tcb.local, tcb.foreign)
				return
			}
		}
		errCh <- err

	default:
		errCh <- fmt.Errorf("connection already exists")
	}
}

func (tcb *TCPpcb) Send(errCh chan error, data []byte) {
	tcpMutex.Lock()
	defer tcpMutex.Unlock()

	switch tcb.state {
	case TCPpcbStateClosed:
		errCh <- fmt.Errorf("connection does not exist")
	case TCPpcbStateListen:
		errCh <- fmt.Errorf("connection does not exist")
	case TCPpcbStateSYNSent, TCPpcbStateSYNReceived:
		errCh <- fmt.Errorf("connection does not exist")
	case TCPpcbStateEstablished, TCPpcbStateCloseWait:
		if len(data) > int(tcb.snd.wnd) {
			errCh <- fmt.Errorf("insufficient resources")
		}

		var err error
		for i := 0; i < 3; i++ { // try to send data at most three time ( because of ARP cache specification of this package).
			if err = TxHelperTCP(tcb, ACK, data, triggerSend, errCh); err != nil {
				log.Printf("[E] TCP SEND call error %s", err.Error())
				time.Sleep(20 * time.Millisecond)
			} else {
				tcb.snd.nxt += uint32(len(data))
				return
			}
		}
		errCh <- err

	default:
		errCh <- fmt.Errorf("connection closing")
	}
}

func (tcb *TCPpcb) Receive(errCh chan error, buf []byte, n *int) {
	tcpMutex.Lock()
	defer tcpMutex.Unlock()

	switch tcb.state {
	case TCPpcbStateClosed:
		errCh <- fmt.Errorf("connection does not exist")
	case TCPpcbStateListen, TCPpcbStateSYNSent, TCPpcbStateSYNReceived:
		errCh <- fmt.Errorf("connection does not exist")
	case TCPpcbStateEstablished, TCPpcbStateFINWait1, TCPpcbStateFINWait2:
		// If insufficient incoming segments are queued to satisfy the
		// request, queue the request.
		if tcb.rcvCmd.errCh != nil {
			errCh <- fmt.Errorf("RECEIVE was already called and data haven't come yet")
			return
		}
		tcb.rcvCmd = rcvCmd{
			n:     n,
			data:  buf,
			errCh: errCh,
		}
		tcb.signalCmd(triggerReceive)
	case TCPpcbStateCloseWait:
		if tcb.rcvCmd.errCh != nil {
			errCh <- fmt.Errorf("RECEIVE was already called and data haven't come yet")
			return
		}
		tcb.rcvCmd = rcvCmd{
			n:     n,
			data:  buf,
			errCh: errCh,
		}
		tcb.signalCmd(triggerReceive)

		// no remaining data
		if tcb.rcvCmd.errCh != nil {
			tcb.rcvCmd = rcvCmd{}
			errCh <- fmt.Errorf("no remaning data")
		}
	default:
		errCh <- fmt.Errorf("connection closing")
	}
}

func (tcb *TCPpcb) Close(errCh chan error) {
	tcpMutex.Lock()
	defer tcpMutex.Unlock()

	switch tcb.state {
	case TCPpcbStateClosed:
		errCh <- fmt.Errorf("connection does not exist")
	case TCPpcbStateListen:
		// Any outstanding RECEIVEs are returned with "error:  closing" responses.
		tcb.signalErr("closing")
		tcb.transition(TCPpcbStateClosed)
		errCh <- nil
	case TCPpcbStateSYNSent:
		// return "error:  closing" responses to any queued SENDs, or RECEIVEs.
		tcb.signalErr("closing")
		tcb.transition(TCPpcbStateClosed)
		errCh <- nil
	case TCPpcbStateSYNReceived:
		// If no SENDs have been issued and there is no pending data to send,
		// then form a FIN segment and send it, and enter FIN-WAIT-1 state;
		// otherwise queue for processing after entering ESTABLISHED state.
		var err error
		for i := 0; i < 3; i++ { // try to send FIN at most three time ( because of ARP cache specification of this package).
			if err = TxHelperTCP(tcb, ACK|FIN, []byte{}, triggerClose, errCh); err != nil {
				log.Printf("[E] TCP CLOSE call error %s", err.Error())
				time.Sleep(20 * time.Millisecond)
			} else {
				tcb.transition(TCPpcbStateFINWait1)
				tcb.snd.nxt++
				log.Printf("[D] active close: local=%s,foreign=%s,closing...", tcb.local, tcb.foreign)
				return
			}
		}
		errCh <- err

	case TCPpcbStateEstablished:
		// Queue this until all preceding SENDs have been segmentized, then
		// form a FIN segment and send it.  In any case, enter FIN-WAIT-1
		// state.
		var err error
		for i := 0; i < 3; i++ { // try to send FIN at most three time ( because of ARP cache specification of this package).
			if err = TxHelperTCP(tcb, ACK|FIN, []byte{}, triggerClose, errCh); err != nil {
				log.Printf("[E] TCP CLOSE call error %s", err.Error())
				time.Sleep(20 * time.Millisecond)
			} else {
				tcb.transition(TCPpcbStateFINWait1)
				tcb.snd.nxt++
				log.Printf("[D] active close: local=%s,foreign=%s,closing...", tcb.local, tcb.foreign)
				return
			}
		}
		errCh <- err

	case TCPpcbStateFINWait1, TCPpcbStateFINWait2:
		errCh <- fmt.Errorf("connection closing")
	case TCPpcbStateCloseWait:
		// Queue this request until all preceding SENDs have been
		// segmentized; then send a FIN segment, enter CLOSING state.
		var err error
		for i := 0; i < 3; i++ { // try to send FIN at most three time ( because of ARP cache specification of this package).
			if err = TxHelperTCP(tcb, ACK|FIN, []byte{}, triggerClose, errCh); err != nil {
				log.Printf("[E] TCP CLOSE call error %s", err.Error())
				time.Sleep(20 * time.Millisecond)
			} else {
				tcb.transition(TCPpcbStateLastACK)
				tcb.snd.nxt++
				return
			}
		}
		errCh <- err

	default:
		errCh <- fmt.Errorf("connection closing")
	}
}

func (tcb *TCPpcb) Abort() error {
	tcpMutex.Lock()
	defer tcpMutex.Unlock()

	switch tcb.state {
	case TCPpcbStateClosed:
		return fmt.Errorf("connection does not exist")
	case TCPpcbStateListen:
		// Any outstanding RECEIVEs should be returned with "error:
		// connection reset" responses
		tcb.signalErr("connection reset")
		tcb.transition(TCPpcbStateClosed)
		return nil
	case TCPpcbStateSYNSent:
		// All queued SENDs and RECEIVEs should be given "connection reset" notification,
		// RECEIVE
		tcb.signalErr("connection reset")
		tcb.transition(TCPpcbStateClosed)
		return nil
	case TCPpcbStateSYNReceived, TCPpcbStateEstablished, TCPpcbStateFINWait1, TCPpcbStateFINWait2, TCPpcbStateCloseWait:
		// All queued SENDs and RECEIVEs should be given "connection reset"
		// notification; all segments queued for transmission (except for the
		// RST formed above) or retransmission should be flushed
		tcb.signalErr("connection reset")
		tcb.transition(TCPpcbStateClosed)
		return TxHandlerTCP(tcb.local, tcb.foreign, []byte{}, tcb.snd.nxt, 0, RST, 0, 0)
	default:
		tcb.transition(TCPpcbStateClosed)
		return nil
	}
}

func (tcb *TCPpcb) Status() TCPpcbState {
	return tcb.state
}
