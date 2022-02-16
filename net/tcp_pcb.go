package net

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"sync"
	"time"
)

/*
	Retransmition Queue entry
*/
const (
	maxRetxCount uint8 = 3
)

type retxEntry struct {
	data      []byte
	seq       uint32
	flag      ControlFlag
	first     time.Time
	last      time.Time
	retxCount uint8
	errCh     []chan error
}

func removeQueue(q []retxEntry, una uint32) []retxEntry {
	index := -1
	for i, entry := range q {

		// not acknowledge yet
		if entry.seq >= una {
			index = i
			break
		}

		// acknowledge and update rto
		// ALPHA=0.9,BETA=1.7
		rtt := time.Since(entry.last)
		srtt = 9*srtt/10 + rtt/10
		srtt = 17 * srtt / 10
		if srtt < lbound {
			rtt = lbound
		} else if srtt > ubound {
			rtt = ubound
		} else {
			rto = srtt
		}
		log.Printf("RTO: %s", rto)

		// notify user
		for _, ch := range entry.errCh {
			ch <- nil
		}
	}
	if index < 0 {
		return q
	}
	return q[index:]
}

/*
	cmd queue entry
*/

const (
	cmdOpen    cmdType = 0
	cmdSend    cmdType = 1
	cmdReceive cmdType = 2
	cmdClose   cmdType = 3
)

type cmdType = uint8

// ReceiveData is used for Receive call
type ReceiveData struct {
	data []byte
	err  error
}

type rcvEntry struct {
	entryTime time.Time
	rcvCh     chan ReceiveData
}

type cmdEntry struct {
	typ       cmdType
	entryTime time.Time
	errCh     chan error
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

// Send Sequence Variables
type snd struct {

	// send unacknowledged
	una uint32

	// send next
	nxt uint32

	// send window
	wnd uint16

	// send urgent pointer
	up uint16

	// segment sequence number used for last window update
	wl1 uint32

	// segment acknowledgment number used for last window update
	wl2 uint32
}

// Receive Sequence Variables
type rcv struct {

	// receive next
	nxt uint32

	// receive window
	wnd uint16

	// receive urgent pointer
	up uint16
}

const (
	bufferSize = math.MaxUint16
)

var (
	tcpMutex sync.Mutex
	tcbs     []*TCPpcb
)

type TCPpcb struct {

	// pcb state
	state TCPpcbState

	// TCP endpoint
	local   TCPEndpoint
	foreign TCPEndpoint

	// Send Sequence Variables
	snd

	// initial send sequence number
	iss uint32

	// Receive Sequence Variables
	rcv

	// initial receive sequence number
	irs uint32

	// maximum segment size
	mss uint16

	// queue
	rxQueue   [bufferSize]byte // receive buffer
	rxLen     uint16
	txQueue   [bufferSize]byte // transmit buffer
	txLen     uint16
	retxQueue []retxEntry // retransmit queue

	// user timeout
	timeout time.Duration

	// user command queue
	cmdQueue []cmdEntry
	rcvQueue []rcvEntry

	// for time-wait state
	lastTxTime time.Time

	// mutex
	mutex sync.Mutex
}

func (tcb *TCPpcb) transition(state TCPpcbState) {
	log.Printf("[I] local=%s, %s => %s", tcb.local, tcb.state, state)
	tcb.state = state
}

func (tcb *TCPpcb) signalErr(msg string) {
	err := fmt.Errorf(msg)
	for _, cmd := range tcb.cmdQueue {
		cmd.errCh <- err
	}
	for _, entry := range tcb.rcvQueue {
		entry.rcvCh <- ReceiveData{
			err: err,
		}
	}
	for _, entry := range tcb.retxQueue {
		for _, ch := range entry.errCh {
			ch <- err
		}
	}
}

func (tcb *TCPpcb) queueFlush(msg string) {
	tcb.signalErr(msg)

	tcb.txLen = 0
	tcb.rxLen = 0
	tcb.cmdQueue = nil
	tcb.rcvQueue = nil
	tcb.retxQueue = nil
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
		rcv:   rcv{wnd: bufferSize},
	}
	tcbs = append(tcbs, tcb)
	return tcb, nil
}

func tcbSelect(address IPAddr, port uint16) *TCPpcb {
	tcpMutex.Lock()
	defer tcpMutex.Unlock()
	for _, t := range tcbs {
		if t.local.Address == address && t.local.Port == port {
			return t
		}
	}
	return nil
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
	tcb.mutex.Lock()
	defer tcb.mutex.Unlock()

	switch tcb.state {
	case TCPpcbStateClosed:
		// passive open
		if !isActive {
			log.Printf("[D] passive open: local=%s,waiting for connection...", tcb.local)
			tcb.timeout = timeout
			tcb.transition(TCPpcbStateListen)
			errCh <- nil
		}
		// active open
		if foreign.Address == IPAddrAny {
			errCh <- fmt.Errorf("foreign socket unspecified")
		}

		iss := createISS()
		var err error
		for i := 0; i < 3; i++ { // try to send SYN at most three time ( because of ARP cache specification of this package).
			if err = TxHandlerTCP(tcb.local, foreign, []byte{}, iss, 0, SYN, tcb.rcv.wnd, 0); err != nil {
				log.Printf("[E] TCP OPEN call error %s", err.Error())
				time.Sleep(20 * time.Millisecond)
			} else {
				tcb.timeout = timeout
				tcb.foreign = foreign

				tcb.iss = iss
				tcb.snd.una = iss
				tcb.snd.nxt = iss + 1
				tcb.transition(TCPpcbStateSYNSent)

				tcb.cmdQueue = append(tcb.cmdQueue, cmdEntry{
					typ:       cmdOpen,
					entryTime: time.Now(),
					errCh:     errCh,
				})
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
		}
		// active open
		if foreign.Address == IPAddrAny {
			errCh <- fmt.Errorf("foreign socket unspecified")
		}

		iss := createISS()
		var err error
		for i := 0; i < 3; i++ { // try to send SYN at most three time ( because of ARP cache specification of this package).
			if err = TxHandlerTCP(tcb.local, foreign, []byte{}, iss, 0, SYN, tcb.rcv.wnd, 0); err != nil {
				log.Printf("[E] TCP OPEN call error %s", err.Error())
				time.Sleep(20 * time.Millisecond)
			} else {
				tcb.timeout = timeout
				tcb.foreign = foreign

				tcb.iss = iss
				tcb.snd.una = iss
				tcb.snd.nxt = iss + 1
				tcb.transition(TCPpcbStateSYNSent)

				tcb.cmdQueue = append(tcb.cmdQueue, cmdEntry{
					typ:       cmdOpen,
					entryTime: time.Now(),
					errCh:     errCh,
				})
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
	tcb.mutex.Lock()
	defer tcb.mutex.Unlock()

	switch tcb.state {
	case TCPpcbStateClosed:
		errCh <- fmt.Errorf("connection does not exist")
	case TCPpcbStateListen:
		if tcb.foreign.Address == IPAddrAny {
			errCh <- fmt.Errorf("foreign socket unspecified")
		}

		iss := createISS()
		tcb.iss = iss
		tcb.snd.una = iss
		tcb.snd.nxt = iss + 1
		tcb.transition(TCPpcbStateSYNSent)

		tcb.cmdQueue = append(tcb.cmdQueue, cmdEntry{
			typ:       cmdSend,
			entryTime: time.Now(),
			errCh:     errCh,
		})
		copy(tcb.txQueue[:], data)
		if err := TxHandlerTCP(tcb.local, tcb.foreign, []byte{}, iss, 0, SYN, tcb.rcv.wnd, 0); err != nil {
			errCh <- err
		}
	case TCPpcbStateSYNSent, TCPpcbStateSYNReceived:
		if int(tcb.txLen)+len(data) >= bufferSize {
			errCh <- fmt.Errorf("insufficient resources")
		}

		tcb.cmdQueue = append(tcb.cmdQueue, cmdEntry{
			typ:       cmdSend,
			entryTime: time.Now(),
			errCh:     errCh,
		})
		copy(tcb.txQueue[tcb.txLen:], data)
	case TCPpcbStateEstablished, TCPpcbStateCloseWait:
		if int(tcb.txLen)+len(data) >= bufferSize {
			errCh <- fmt.Errorf("insufficient resources")
		}

		if tcb.txLen == 0 {
			errCh <- nil
		}

		copy(tcb.txQueue[tcb.txLen:], data)
		tcb.txLen += uint16(len(data))
		err := TxHandlerTCP(tcb.local, tcb.foreign, tcb.txQueue[:tcb.txLen], tcb.snd.nxt, tcb.rcv.nxt, ACK, tcb.rcv.wnd, 0)

		// error
		if err != nil {
			// delete SEND user call
			var deleteIndex []int
			for i, entry := range tcb.cmdQueue {
				if entry.typ == cmdSend {
					entry.errCh <- err
					deleteIndex = append(deleteIndex, i)
				}
			}
			tcb.cmdQueue = removeCmd(tcb.cmdQueue, deleteIndex)

			tcb.txLen = 0
			errCh <- err
			return
		}

		// there is no error
		var deleteIndex []int
		var errChs []chan error
		for i, entry := range tcb.cmdQueue {
			if entry.typ == cmdSend {
				deleteIndex = append(deleteIndex, i)
				errChs = append(errChs, entry.errCh)
			}
		}
		tcb.cmdQueue = removeCmd(tcb.cmdQueue, deleteIndex)
		errChs = append(errChs, errCh)

		// if transmit is sucessful,push data to retransmitQueue.
		tcb.retxQueue = append(tcb.retxQueue, retxEntry{
			data:  tcb.txQueue[:tcb.txLen],
			seq:   tcb.snd.nxt,
			flag:  ACK,
			first: time.Now(),
			last:  time.Now(),
			errCh: errChs,
		})
		tcb.snd.nxt += uint32(tcb.txLen) // TODO:window size?
		tcb.txLen = 0

	default:
		errCh <- fmt.Errorf("connection closing")
	}
}

func (tcb *TCPpcb) Receive(rcvCh chan ReceiveData) {
	tcb.mutex.Lock()
	defer tcb.mutex.Unlock()

	switch tcb.state {
	case TCPpcbStateClosed:
		rcvCh <- ReceiveData{
			err: fmt.Errorf("connection does not exist"),
		}
	case TCPpcbStateListen, TCPpcbStateSYNSent, TCPpcbStateSYNReceived:
		tcb.rcvQueue = append(tcb.rcvQueue, rcvEntry{
			entryTime: time.Now(),
			rcvCh:     rcvCh,
		})
	case TCPpcbStateEstablished, TCPpcbStateFINWait1, TCPpcbStateFINWait2:
		// TODO:
		// If insufficient incoming segments are queued to satisfy the
		// request, queue the request.
		rcvCh <- ReceiveData{
			data: tcb.rxQueue[:tcb.rxLen],
		}
		tcb.rxLen = 0
	case TCPpcbStateCloseWait:
		// no remaining data
		if tcb.rxLen == 0 {
			rcvCh <- ReceiveData{
				err: fmt.Errorf("connection closing"),
			}
		}
		// remaining data
		rcvCh <- ReceiveData{
			data: tcb.rxQueue[:tcb.rxLen],
		}
		tcb.rxLen = 0
	default:
		rcvCh <- ReceiveData{
			err: fmt.Errorf("connection closing"),
		}
	}
}

func (tcb *TCPpcb) Close(errCh chan error) {
	tcb.mutex.Lock()
	defer tcb.mutex.Unlock()

	switch tcb.state {
	case TCPpcbStateClosed:
		errCh <- fmt.Errorf("connection does not exist")
	case TCPpcbStateListen:
		// Any outstanding RECEIVEs are returned with "error:  closing" responses.
		for _, rcv := range tcb.rcvQueue {
			rcv.rcvCh <- ReceiveData{
				err: fmt.Errorf("closing responses"),
			}
		}
		tcb.rcvQueue = nil
		tcb.transition(TCPpcbStateClosed)
		errCh <- nil
	case TCPpcbStateSYNSent:
		// return "error:  closing" responses to any queued SENDs, or RECEIVEs.
		// RECEIVE
		for _, rcv := range tcb.rcvQueue {
			rcv.rcvCh <- ReceiveData{
				err: fmt.Errorf("closing responses"),
			}
		}
		tcb.rcvQueue = nil
		// SEND
		var deleteIndex []int
		for i, entry := range tcb.cmdQueue {
			if entry.typ == cmdSend {
				entry.errCh <- fmt.Errorf("closing responses")
				deleteIndex = append(deleteIndex, i)
			}
		}
		tcb.cmdQueue = removeCmd(tcb.cmdQueue, deleteIndex)
		tcb.transition(TCPpcbStateClosed)

		errCh <- nil
	case TCPpcbStateSYNReceived:
		// If no SENDs have been issued and there is no pending data to send,
		// then form a FIN segment and send it, and enter FIN-WAIT-1 state;
		// otherwise queue for processing after entering ESTABLISHED state.
		finished := true
		for _, entry := range tcb.cmdQueue {
			if entry.typ == cmdSend {
				finished = false
			}
		}

		if finished {
			var err error
			for i := 0; i < 3; i++ { // try to send FIN at most three time ( because of ARP cache specification of this package).
				if err = TxHandlerTCP(tcb.local, tcb.foreign, []byte{}, tcb.snd.nxt, tcb.rcv.nxt, FIN, tcb.rcv.wnd, 0); err != nil {
					time.Sleep(20 * time.Millisecond)
				} else {
					tcb.transition(TCPpcbStateFINWait1)

					tcb.cmdQueue = append(tcb.cmdQueue, cmdEntry{
						typ:       cmdClose,
						entryTime: time.Now(),
						errCh:     errCh,
					})
					log.Printf("[D] active close: local=%s,foreign=%s,closing...", tcb.local, tcb.foreign)
					return
				}
			}
			errCh <- err
		} else {
			tcb.cmdQueue = append(tcb.cmdQueue, cmdEntry{
				typ:       cmdClose,
				entryTime: time.Now(),
				errCh:     errCh,
			})
		}
	case TCPpcbStateEstablished:
		// Queue this until all preceding SENDs have been segmentized, then
		// form a FIN segment and send it.  In any case, enter FIN-WAIT-1
		// state.
		var err error
		if tcb.txLen > 0 {
			err = TxHandlerTCP(tcb.local, tcb.foreign, tcb.txQueue[:tcb.txLen], tcb.snd.nxt, tcb.rcv.nxt, ACK, tcb.rcv.wnd, 0)

			if err != nil {
				// delete SEND user call
				var deleteIndex []int
				for i, entry := range tcb.cmdQueue {
					if entry.typ == cmdSend {
						entry.errCh <- err
						deleteIndex = append(deleteIndex, i)
					}
				}
				tcb.cmdQueue = removeCmd(tcb.cmdQueue, deleteIndex)

				tcb.txLen = 0
			} else {
				var deleteIndex []int
				var errChs []chan error
				for i, entry := range tcb.cmdQueue {
					if entry.typ == cmdSend {
						deleteIndex = append(deleteIndex, i)
						errChs = append(errChs, entry.errCh)
					}
				}
				tcb.cmdQueue = removeCmd(tcb.cmdQueue, deleteIndex)
				errChs = append(errChs, errCh)

				// if transmit is sucessful,push data to retransmitQueue.
				tcb.retxQueue = append(tcb.retxQueue, retxEntry{
					data:  tcb.txQueue[:tcb.txLen],
					seq:   tcb.snd.nxt,
					flag:  ACK,
					first: time.Now(),
					last:  time.Now(),
					errCh: errChs,
				})
				tcb.snd.nxt += uint32(tcb.txLen) // TODO:window size?

				tcb.txLen = 0
			}
		}

		if err = TxHandlerTCP(tcb.local, tcb.foreign, []byte{}, tcb.snd.nxt, tcb.rcv.nxt, FIN, tcb.rcv.wnd, 0); err != nil {
			errCh <- err
		} else {
			tcb.transition(TCPpcbStateFINWait1)

			tcb.cmdQueue = append(tcb.cmdQueue, cmdEntry{
				typ:       cmdClose,
				entryTime: time.Now(),
				errCh:     errCh,
			})
			log.Printf("[D] active close: local=%s,foreign=%s,closing...", tcb.local, tcb.foreign)
			return
		}

	case TCPpcbStateFINWait1, TCPpcbStateFINWait2:
		errCh <- fmt.Errorf("connection closing")
	case TCPpcbStateCloseWait:
		// Queue this request until all preceding SENDs have been
		// segmentized; then send a FIN segment, enter CLOSING state.
		var err error
		if tcb.txLen > 0 {
			err = TxHandlerTCP(tcb.local, tcb.foreign, tcb.txQueue[:tcb.txLen], tcb.snd.nxt, tcb.rcv.nxt, ACK, tcb.rcv.wnd, 0)

			if err != nil {
				// delete SEND user call
				var deleteIndex []int
				for i, entry := range tcb.cmdQueue {
					if entry.typ == cmdSend {
						entry.errCh <- err
						deleteIndex = append(deleteIndex, i)
					}
				}
				tcb.cmdQueue = removeCmd(tcb.cmdQueue, deleteIndex)

				tcb.txLen = 0
			} else {
				var deleteIndex []int
				var errChs []chan error
				for i, entry := range tcb.cmdQueue {
					if entry.typ == cmdSend {
						deleteIndex = append(deleteIndex, i)
						errChs = append(errChs, entry.errCh)
					}
				}
				tcb.cmdQueue = removeCmd(tcb.cmdQueue, deleteIndex)
				errChs = append(errChs, errCh)

				// if transmit is sucessful,push data to retransmitQueue.
				tcb.retxQueue = append(tcb.retxQueue, retxEntry{
					data:  tcb.txQueue[:tcb.txLen],
					seq:   tcb.snd.nxt,
					flag:  ACK,
					first: time.Now(),
					last:  time.Now(),
					errCh: errChs,
				})
				tcb.snd.nxt += uint32(tcb.txLen) // TODO:window size?

				tcb.txLen = 0
			}
		}

		if err = TxHandlerTCP(tcb.local, tcb.foreign, []byte{}, tcb.snd.nxt, tcb.rcv.nxt, FIN, tcb.rcv.wnd, 0); err != nil {
			errCh <- err
		} else {
			tcb.transition(TCPpcbStateClosing)

			tcb.cmdQueue = append(tcb.cmdQueue, cmdEntry{
				typ:       cmdClose,
				entryTime: time.Now(),
				errCh:     errCh,
			})
			log.Printf("[D] active close: local=%s,foreign=%s,closing...", tcb.local, tcb.foreign)
			return
		}

	default:
		errCh <- fmt.Errorf("connection closing")
	}
}

func (tcb *TCPpcb) Abort() error {
	tcb.mutex.Lock()
	defer tcb.mutex.Unlock()

	switch tcb.state {
	case TCPpcbStateClosed:
		return fmt.Errorf("connection does not exist")
	case TCPpcbStateListen:
		// Any outstanding RECEIVEs should be returned with "error:
		// connection reset" responses
		for _, rcv := range tcb.rcvQueue {
			rcv.rcvCh <- ReceiveData{
				err: fmt.Errorf("closing reset"),
			}
		}
		tcb.rcvQueue = nil

		tcb.transition(TCPpcbStateClosed)
		return nil
	case TCPpcbStateSYNSent:
		// All queued SENDs and RECEIVEs should be given "connection reset" notification,
		// RECEIVE
		for _, rcv := range tcb.rcvQueue {
			rcv.rcvCh <- ReceiveData{
				err: fmt.Errorf("closing reset"),
			}
		}
		tcb.rcvQueue = nil
		// SEND
		var deleteIndex []int
		for i, entry := range tcb.cmdQueue {
			if entry.typ == cmdSend {
				entry.errCh <- fmt.Errorf("closing reset")
				deleteIndex = append(deleteIndex, i)
			}
		}
		tcb.cmdQueue = removeCmd(tcb.cmdQueue, deleteIndex)
		tcb.transition(TCPpcbStateClosed)
		return nil
	case TCPpcbStateSYNReceived, TCPpcbStateEstablished, TCPpcbStateFINWait1, TCPpcbStateFINWait2, TCPpcbStateCloseWait:
		// All queued SENDs and RECEIVEs should be given "connection reset"
		// notification; all segments queued for transmission (except for the
		// RST formed above) or retransmission should be flushed
		// RECEIVE
		for _, rcv := range tcb.rcvQueue {
			rcv.rcvCh <- ReceiveData{
				err: fmt.Errorf("closing reset"),
			}
		}
		tcb.rcvQueue = nil
		// SEND
		var deleteIndex []int
		for i, entry := range tcb.cmdQueue {
			if entry.typ == cmdSend {
				entry.errCh <- fmt.Errorf("closing reset")
				deleteIndex = append(deleteIndex, i)
			}
		}
		tcb.cmdQueue = removeCmd(tcb.cmdQueue, deleteIndex)
		tcb.transition(TCPpcbStateClosed)
		return TxHandlerTCP(tcb.local, tcb.foreign, []byte{}, tcb.snd.nxt, 0, RST, 0, 0)
	default:
		tcb.transition(TCPpcbStateClosed)
		return nil
	}
}

func (tcb *TCPpcb) Status() (string, error) {
	tcb.mutex.Lock()
	defer tcb.mutex.Unlock()

	switch tcb.state {
	case TCPpcbStateClosed:
		return "", fmt.Errorf("connection does not exist")
	default:
		return fmt.Sprintf("state = %s", tcb.state), nil
	}
}
