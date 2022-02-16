package net

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"math/rand"
	"sync"
	"time"
)

func TCPInit(done chan struct{}) error {
	go tcpTimer(done)
	rand.Seed(time.Now().UnixNano())
	return IPProtocolRegister(&TCPProtocol{})
}

/*
	TCP endpoint
*/

type TCPEndpoint = UDPEndpoint

// Str2TCPEndpoint encodes str to TCPEndpoint
// ex) str="8.8.8.8:80"
func Str2TCPEndpoint(str string) (TCPEndpoint, error) {
	return Str2UDPEndpoint(str)
}

/*
	TCP Header
*/

const (
	CWR ControlFlag = 0b10000000
	ECE ControlFlag = 0b01000000
	URG ControlFlag = 0b00100000
	ACK ControlFlag = 0b00010000
	PSH ControlFlag = 0b00001000
	RST ControlFlag = 0b00000100
	SYN ControlFlag = 0b00000010
	FIN ControlFlag = 0b00000001
)

type ControlFlag uint8

func up(a ControlFlag, b ControlFlag, str string) string {
	if a&b > 0 {
		return str
	}
	return ""
}

func (f ControlFlag) String() string {
	return fmt.Sprintf(
		"%s%s%s%s%s%s%s%s",
		up(f, CWR, "CWR "),
		up(f, ECE, "ECE "),
		up(f, URG, "URG "),
		up(f, ACK, "ACK "),
		up(f, PSH, "PSH "),
		up(f, RST, "RST "),
		up(f, SYN, "SYN "),
		up(f, FIN, "FIN "),
	)
}

const (
	TCPHeaderSizeMin    = 20
	TCPPseudoHeaderSize = 12
)

// TCPHeader is header for TCP protocol
type TCPHeader struct {

	// source port number
	Src uint16

	// destination port number
	Dst uint16

	// sequence number
	Seq uint32

	// acknowledgement number
	Ack uint32

	// Offset is assembly of data offset(4bit) and reserved bit(4bit)
	Offset uint8

	// control flag
	Flag ControlFlag

	// window size
	Window uint16

	// checksum
	Checksum uint16

	// urgent pointer
	Urgent uint16
}

func (h TCPHeader) String() string {
	return fmt.Sprintf(`
		Dst: %d, 
		Src: %d,
		Seq: %d, 
		Ack: %d,
		Offset: %d,
		Control Flag: %s,
		Window Size: %d,
		Checksum: %x,
		Urgent Pointer: %x,
	`, h.Dst, h.Src, h.Seq, h.Ack, h.Offset>>4, h.Flag, h.Window, h.Checksum, h.Urgent)
}

// TCPPseudoHeader is used for caluculating checksum.
type TCPPseudoHeader struct {

	// source IP address
	Src IPAddr

	// destination IP address
	Dst IPAddr

	// padding, always 0
	Pad uint8

	// TCP protocol type,always 6
	Type IPProtocolType

	// length of tcp packet
	Len uint16
}

// data2headerTCP transforms data to TCP header.
// returned []byte contains Options
// src,dst is used for caluculating checksum.
func data2headerTCP(data []byte, src IPAddr, dst IPAddr) (TCPHeader, []byte, error) {

	if len(data) < TCPHeaderSizeMin {
		return TCPHeader{}, nil, fmt.Errorf("data size is too small for TCP Header")
	}

	// read header in bigEndian
	var hdr TCPHeader
	r := bytes.NewReader(data)
	err := binary.Read(r, binary.BigEndian, &hdr)
	if err != nil {
		return TCPHeader{}, nil, err
	}

	// caluculate checksum
	pseudoHdr := TCPPseudoHeader{
		Src:  src,
		Dst:  dst,
		Type: IPProtocolTCP,
		Len:  uint16(len(data)),
	}
	var w bytes.Buffer
	err = binary.Write(&w, binary.BigEndian, pseudoHdr)
	if err != nil {
		return TCPHeader{}, nil, err
	}
	chksum := CheckSum(w.Bytes(), 0)
	chksum = CheckSum(data, uint32(^chksum))
	if chksum != 0 && chksum != 0xffff {
		return TCPHeader{}, nil, fmt.Errorf("checksum error (TCP)")
	}

	return hdr, data[TCPHeaderSizeMin:], nil
}

func header2dataTCP(hdr *TCPHeader, payload []byte, src IPAddr, dst IPAddr) ([]byte, error) {

	// pseudo header for caluculating checksum afterwards
	pseudoHdr := TCPPseudoHeader{
		Src:  src,
		Dst:  dst,
		Type: IPProtocolTCP,
		Len:  uint16(TCPHeaderSizeMin + len(payload)),
	}

	// write header in bigEndian
	var w bytes.Buffer
	err := binary.Write(&w, binary.BigEndian, pseudoHdr)
	if err != nil {
		return nil, err
	}
	err = binary.Write(&w, binary.BigEndian, hdr)
	if err != nil {
		return nil, err
	}

	// write payload as it is
	_, err = w.Write(payload)
	if err != nil {
		return nil, err
	}

	// caluculate checksum
	buf := w.Bytes()
	chksum := CheckSum(buf, 0)
	copy(buf[28:30], Hton16(chksum)) // considering TCPPseudoHeaderSize

	// set checksum in the header (for debug)
	hdr.Checksum = chksum
	return buf[TCPPseudoHeaderSize:], nil
}

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
	txTime    time.Time
	retxCount uint8
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
		rtt := time.Since(entry.txTime)
		srtt = 9*srtt/10 + rtt/10
		srtt = 17 * srtt / 10
		if srtt < lbound {
			rtt = lbound
		} else if srtt > ubound {
			rtt = ubound
		} else {
			rto = srtt
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
	haveSent  bool // if typ == cmdSend,the data of the entry have been sent or not
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

	// user command queue
	cmdQueue []cmdEntry
	rcvQueue []rcvEntry
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

	switch tcb.state {
	case TCPpcbStateClosed:
		// passive open
		if !isActive {
			tcb.state = TCPpcbStateListen
			errCh <- nil
		}
		// active open
		if foreign.Address == IPAddrAny {
			errCh <- fmt.Errorf("foreign socket unspecified")
		}

		tcb.foreign = foreign
		iss := createISS()
		tcb.iss = iss
		tcb.snd.una = iss
		tcb.snd.nxt = iss + 1
		tcb.state = TCPpcbStateSYNSent

		tcb.cmdQueue = append(tcb.cmdQueue, cmdEntry{
			typ:       cmdOpen,
			entryTime: time.Now(),
			errCh:     errCh,
		})
		if err := TxHandlerTCP(tcb.local, foreign, []byte{}, iss, 0, SYN, tcb.rcv.wnd, 0); err != nil {
			errCh <- err
		}
	case TCPpcbStateListen:
		// passive open
		if !isActive {
			errCh <- nil
		}
		// active open
		if foreign.Address == IPAddrAny {
			errCh <- fmt.Errorf("foreign socket unspecified")
		}

		tcb.foreign = foreign
		iss := createISS()
		tcb.iss = iss
		tcb.snd.una = iss
		tcb.snd.nxt = iss + 1
		tcb.state = TCPpcbStateSYNSent

		tcb.cmdQueue = append(tcb.cmdQueue, cmdEntry{
			typ:       cmdOpen,
			entryTime: time.Now(),
			errCh:     errCh,
		})
		if err := TxHandlerTCP(tcb.local, foreign, []byte{}, iss, 0, SYN, tcb.rcv.wnd, 0); err != nil {
			errCh <- err
		}
	default:
		errCh <- fmt.Errorf("connection already exists")
	}
}

func (tcb *TCPpcb) Send(errCh chan error, data []byte) {
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
		tcb.state = TCPpcbStateSYNSent

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
		// change state of SEND user call
		for _, entry := range tcb.cmdQueue {
			if entry.typ == cmdSend {
				entry.haveSent = true
			}
		}
		tcb.cmdQueue = append(tcb.cmdQueue, cmdEntry{
			typ:       cmdSend,
			entryTime: time.Now(),
			haveSent:  true,
			errCh:     errCh,
		})

		// if transmit is sucessful,push data to retransmitQueue.
		tcb.retxQueue = append(tcb.retxQueue, retxEntry{
			data: tcb.txQueue[:tcb.txLen],
			seq:  tcb.snd.nxt,
			flag: ACK,
		})
		tcb.snd.nxt += uint32(tcb.txLen) // TODO:window size?
		tcb.txLen = 0

	default:
		errCh <- fmt.Errorf("connection closing")
	}
}

func (tcb *TCPpcb) Receive(rcvCh chan ReceiveData) {
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
		tcb.state = TCPpcbStateClosed
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

		tcb.state = TCPpcbStateClosed
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
			tcb.state = TCPpcbStateFINWait1
			errCh <- TxHandlerTCP(tcb.local, tcb.foreign, []byte{}, tcb.snd.nxt, tcb.rcv.nxt, FIN, tcb.rcv.wnd, 0)
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
		err := TxHandlerTCP(tcb.local, tcb.foreign, tcb.txQueue[:tcb.txLen], tcb.snd.nxt, tcb.rcv.nxt, ACK, tcb.rcv.wnd, 0)

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
			// change state of SEND user call
			for _, entry := range tcb.cmdQueue {
				if entry.typ == cmdSend {
					entry.haveSent = true
				}
			}

			// if transmit is sucessful,push data to retransmitQueue.
			tcb.retxQueue = append(tcb.retxQueue, retxEntry{
				data: tcb.txQueue[:tcb.txLen],
				seq:  tcb.snd.nxt,
				flag: ACK,
			})
			tcb.snd.nxt += uint32(tcb.txLen) // TODO:window size?

			tcb.txLen = 0
		}

		tcb.state = TCPpcbStateFINWait1
		errCh <- TxHandlerTCP(tcb.local, tcb.foreign, []byte{}, tcb.snd.nxt, tcb.rcv.nxt, FIN, tcb.rcv.wnd, 0)
	case TCPpcbStateFINWait1, TCPpcbStateFINWait2:
		errCh <- fmt.Errorf("connection closing")
	case TCPpcbStateCloseWait:
		// TODO:
		// Queue this request until all preceding SENDs have been
		// segmentized; then send a FIN segment, enter CLOSING state.
		err := TxHandlerTCP(tcb.local, tcb.foreign, tcb.txQueue[:tcb.txLen], tcb.snd.nxt, tcb.rcv.nxt, ACK, tcb.rcv.wnd, 0)

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
			// change state of SEND user call
			for _, entry := range tcb.cmdQueue {
				if entry.typ == cmdSend {
					entry.haveSent = true
				}
			}

			// if transmit is sucessful,push data to retransmitQueue.
			tcb.retxQueue = append(tcb.retxQueue, retxEntry{
				data: tcb.txQueue[:tcb.txLen],
				seq:  tcb.snd.nxt,
				flag: ACK,
			})
			tcb.snd.nxt += uint32(tcb.txLen) // TODO:window size?

			tcb.txLen = 0
		}

		tcb.state = TCPpcbStateClosing
		errCh <- TxHandlerTCP(tcb.local, tcb.foreign, []byte{}, tcb.snd.nxt, tcb.rcv.nxt, FIN, tcb.rcv.wnd, 0)
	default:
		errCh <- fmt.Errorf("connection closing")
	}
}

func (tcb *TCPpcb) Abort() error {
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

		tcb.state = TCPpcbStateClosed
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

		tcb.state = TCPpcbStateClosed
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

		tcb.state = TCPpcbStateClosed
		return TxHandlerTCP(tcb.local, tcb.foreign, []byte{}, tcb.snd.nxt, 0, RST, 0, 0)
	default:
		tcb.state = TCPpcbStateClosed
		return nil
	}
}

func (tcb *TCPpcb) Status() (string, error) {
	switch tcb.state {
	case TCPpcbStateClosed:
		return "", fmt.Errorf("connection does not exist")
	default:
		return fmt.Sprintf("state = %s", tcb.state), nil
	}
}

/*
	TCP Protocol
*/
// TCPProtocol is struct for TCP protocol handler.
// This implements IPUpperProtocol interface.
type TCPProtocol struct{}

func (p *TCPProtocol) Type() IPProtocolType {
	return IPProtocolTCP
}

func (p *TCPProtocol) RxHandler(data []byte, src IPAddr, dst IPAddr, ipIface *IPIface) error {

	hdr, payload, err := data2headerTCP(data, src, dst)
	if err != nil {
		return err
	}
	log.Printf("[D] TCP RxHandler: src=%s:%d,dst=%s:%d,iface=%s,tcp header=%s,payload=%v", src, hdr.Src, dst, hdr.Dst, ipIface.Family(), hdr, payload)

	// search TCP pcb
	tcb := tcbSelect(dst, hdr.Dst)
	if tcb == nil {
		return fmt.Errorf("destination TCP protocol control block not found")
	}

	return segmentArrives(tcb, hdr, payload, uint32(len(payload)), src)
}

func segmentArrives(tcb *TCPpcb, hdr TCPHeader, data []byte, dataLen uint32, src IPAddr) error {
	switch tcb.state {
	case TCPpcbStateClosed:
		if hdr.Flag&RST > 0 {
			// An incoming segment containing a RST is discarded
			return nil
		}
		// ACK bit is off
		if hdr.Flag&ACK == 0 {
			return TxHandlerTCP(tcb.local, tcb.foreign, []byte{}, 0, hdr.Seq+dataLen, RST|ACK, 0, 0)
		}
		// ACK bit is on
		return TxHandlerTCP(tcb.local, tcb.foreign, []byte{}, hdr.Ack, 0, RST, 0, 0)

	case TCPpcbStateListen:
		// first check for an RST
		if hdr.Flag&RST > 0 {
			// An incoming RST should be ignored
			return nil
		}

		// second check for an ACK
		if hdr.Flag&ACK > 0 {
			// Any acknowledgment is bad if it arrives on a connection still in the LISTEN state.
			// An acceptable reset segment should be formed for any arriving ACK-bearing segment.
			return TxHandlerTCP(tcb.local, tcb.foreign, []byte{}, hdr.Ack, 0, RST, 0, 0)
		}

		// third check for a SYN
		if hdr.Flag&SYN > 0 {
			// ignore security check

			tcb.rcv.nxt = hdr.Seq + 1
			tcb.irs = hdr.Seq

			tcb.iss = createISS()
			tcb.snd.nxt = tcb.iss + 1
			tcb.snd.una = tcb.iss
			tcb.state = TCPpcbStateSYNReceived

			tcb.foreign = TCPEndpoint{
				Address: src,
				Port:    hdr.Src,
			}

			copy(tcb.rxQueue[tcb.rxLen:], data)
			tcb.rxLen += uint16(dataLen)
			return TxHandlerTCP(tcb.local, tcb.foreign, []byte{}, tcb.iss, tcb.rcv.nxt, SYN|ACK, 0, 0)
		}

		// fourth other text or control
		log.Printf("[D] TCP segment discarded, tcp header=%s", hdr)
		return nil

	case TCPpcbStateSYNSent:
		// first check the ACK bit
		var acceptable bool
		if hdr.Flag&ACK > 0 {
			// If SEG.ACK =< ISS, or SEG.ACK > SND.NXT, send a reset (unless
			// the RST bit is set, if so drop the segment and return)
			if hdr.Ack <= tcb.iss || hdr.Ack > tcb.snd.nxt {
				if hdr.Flag&RST > 0 {
					log.Printf("[D] TCP segment discarded, tcp header=%s", hdr)
					return nil
				}
				return TxHandlerTCP(tcb.local, tcb.foreign, []byte{}, hdr.Ack, 0, RST, 0, 0)
			}
			if tcb.snd.una <= hdr.Ack && hdr.Ack <= tcb.snd.nxt {
				// this ACK is  acceptable
				acceptable = true
			} else {
				log.Printf("[I] ACK is not acceptable")
			}
		}

		// second check the RST bit
		if hdr.Flag&RST > 0 {
			if acceptable {
				tcb.state = TCPpcbStateClosed
				return fmt.Errorf("connection reset")
			}
			log.Printf("[D] TCP segment discarded, tcp header=%s", hdr)
			return nil
		}

		// third check the security and precedence
		// ignore

		// fourth check the SYN bit
		if hdr.Flag&SYN > 0 {
			tcb.rcv.nxt = hdr.Seq + 1
			tcb.irs = hdr.Seq

			if hdr.Flag&ACK > 0 { // our SYN has been ACKed
				tcb.snd.una = hdr.Ack
				tcb.retxQueue = removeQueue(tcb.retxQueue, tcb.snd.una)
			}

			if tcb.snd.una > tcb.una {
				tcb.state = TCPpcbStateEstablished
				err := TxHandlerTCP(tcb.local, tcb.foreign, tcb.txQueue[:tcb.txLen], tcb.snd.nxt, tcb.rcv.nxt, ACK, tcb.rcv.wnd, 0)

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
					// change state of SEND user call
					for _, entry := range tcb.cmdQueue {
						if entry.typ == cmdSend {
							entry.haveSent = true
						}
					}

					// if transmit is sucessful,push data to retransmitQueue.
					tcb.retxQueue = append(tcb.retxQueue, retxEntry{
						data: tcb.txQueue[:tcb.txLen],
						seq:  tcb.snd.nxt,
						flag: ACK,
					})
					tcb.snd.nxt += uint32(tcb.txLen) // TODO:window size?

					tcb.txLen = 0
				}

				return err
			}

			tcb.state = TCPpcbStateSYNReceived
			// TODO:
			// If there are other controls or text in the
			// segment, queue them for processing after the ESTABLISHED state
			// has been reached
			return TxHandlerTCP(tcb.local, tcb.foreign, []byte{}, tcb.iss, tcb.rcv.nxt, SYN|ACK, bufferSize-tcb.rxLen, 0)
		}

		// fifth, if neither of the SYN or RST bits is set then drop the segment and return.
		log.Printf("[D] TCP segment discarded tcp header=%s", hdr)
		return nil

	default:
		// first check sequence number
		var acceptable bool
		if tcb.rcv.wnd == 0 {
			if dataLen == 0 && hdr.Seq == tcb.rcv.nxt {
				acceptable = true
			}
			if dataLen > 0 {
				acceptable = true
			}
		} else {
			if dataLen == 0 && (tcb.rcv.nxt <= hdr.Seq && hdr.Seq < tcb.rcv.nxt+uint32(tcb.rcv.wnd)) {
				acceptable = true
			}
			if dataLen > 0 && (tcb.rcv.nxt <= hdr.Seq || hdr.Seq < tcb.rcv.nxt+uint32(tcb.rcv.wnd)) || (tcb.rcv.nxt <= hdr.Seq+dataLen-1 && hdr.Seq+dataLen-1 < tcb.rcv.nxt+uint32(tcb.rcv.wnd)) {
				acceptable = true
			}
		}
		if !acceptable {
			if hdr.Flag&RST > 0 {
				return nil
			}
			return TxHandlerTCP(tcb.local, tcb.foreign, []byte{}, tcb.snd.nxt, tcb.rcv.nxt, ACK, bufferSize-tcb.rxLen, 0)
		}
		// In the following it is assumed that the segment is the idealized
		// segment that begins at RCV.NXT and does not exceed the window.
		data = data[tcb.rcv.nxt-hdr.Seq : tcb.rcv.nxt+uint32(tcb.rcv.wnd)-hdr.Seq] // TODO:correct window size

		// second check the RST bit
		switch tcb.state {
		case TCPpcbStateSYNReceived:
			if hdr.Flag&RST > 0 {
				// If this connection was initiated with an active OPEN (i.e., came
				// from SYN-SENT state) then the connection was refused, signal
				// the user "connection refused".
				var deleteIndex []int
				for i, cmd := range tcb.cmdQueue {
					if cmd.typ == cmdOpen {
						cmd.errCh <- fmt.Errorf("connection refused")
						deleteIndex = append(deleteIndex, i)
					}
				}
				tcb.cmdQueue = removeCmd(tcb.cmdQueue, deleteIndex)

				tcb.retxQueue = nil
				tcb.state = TCPpcbStateClosed
				return nil
			}
		case TCPpcbStateEstablished, TCPpcbStateFINWait1, TCPpcbStateFINWait2, TCPpcbStateCloseWait:
			if hdr.Flag&RST > 0 {
				// any outstanding RECEIVEs and SEND should receive "reset" responses
				// RECEIVE
				for _, rcv := range tcb.rcvQueue {
					rcv.rcvCh <- ReceiveData{
						err: fmt.Errorf("reset"),
					}
				}
				tcb.rcvQueue = nil
				// SEND
				var deleteIndex []int
				for i, entry := range tcb.cmdQueue {
					if entry.typ == cmdSend {
						entry.errCh <- fmt.Errorf("reset")
						deleteIndex = append(deleteIndex, i)
					}
				}
				tcb.cmdQueue = removeCmd(tcb.cmdQueue, deleteIndex)

				tcb.txLen = 0
				tcb.rxLen = 0
				tcb.retxQueue = nil
				// TODO:
				// Users should also receive an unsolicited general "connection reset" signal
				// there is no way to notify users (ex interrupt)
				tcb.state = TCPpcbStateClosed
				return nil
			}
		case TCPpcbStateClosing, TCPpcbStateLastACK, TCPpcbStateTimeWait:
			if hdr.Flag&RST > 0 {
				tcb.state = TCPpcbStateClosed
				return nil
			}
		}

		// third check security and precedence
		// ignore

		// fourth, check the SYN bit
		switch tcb.state {
		case TCPpcbStateSYNReceived, TCPpcbStateEstablished, TCPpcbStateFINWait1, TCPpcbStateFINWait2,
			TCPpcbStateCloseWait, TCPpcbStateClosing, TCPpcbStateLastACK, TCPpcbStateTimeWait:
			if hdr.Flag&SYN > 0 {
				// any outstanding RECEIVEs and SEND should receive "reset" responses,
				// all segment queues should be flushed, the user should also
				// receive an unsolicited general "connection reset" signal
				// RECEIVE
				for _, rcv := range tcb.rcvQueue {
					rcv.rcvCh <- ReceiveData{
						err: fmt.Errorf("reset"),
					}
				}
				tcb.rcvQueue = nil
				// SEND
				var deleteIndex []int
				for i, entry := range tcb.cmdQueue {
					if entry.typ == cmdSend {
						entry.errCh <- fmt.Errorf("reset")
						deleteIndex = append(deleteIndex, i)
					}
				}
				tcb.cmdQueue = removeCmd(tcb.cmdQueue, deleteIndex)

				tcb.txLen = 0
				tcb.rxLen = 0
				tcb.retxQueue = nil

				tcb.state = TCPpcbStateClosed
				return TxHandlerTCP(tcb.local, tcb.foreign, []byte{}, tcb.snd.nxt, tcb.rcv.nxt, RST, bufferSize-tcb.rxLen, 0)
			}
		}

		// fifth check the ACK field
		if hdr.Flag&ACK == 0 {
			log.Printf("[D] TCP segment discarded tcp header=%s", hdr)
			return nil
		}
		switch tcb.state {
		case TCPpcbStateSYNReceived:
			if tcb.snd.una <= hdr.Ack && hdr.Ack <= tcb.snd.nxt {
				tcb.state = TCPpcbStateEstablished
			} else {
				log.Printf("unacceptable ACK is sent")
				return TxHandlerTCP(tcb.local, tcb.foreign, []byte{}, hdr.Ack, 0, RST, 0, 0)
			}
		case TCPpcbStateEstablished, TCPpcbStateFINWait1, TCPpcbStateFINWait2, TCPpcbStateCloseWait, TCPpcbStateClosing:
			if tcb.snd.una < hdr.Ack && hdr.Ack <= tcb.snd.nxt {
				tcb.snd.una = hdr.Ack
				tcb.retxQueue = removeQueue(tcb.retxQueue, tcb.snd.una)

				// Users should receive
				// positive acknowledgments for buffers which have been SENT and
				// fully acknowledged (i.e., SEND buffer should be returned with
				// "ok" response)
				var deleteIndex []int
				for i, entry := range tcb.cmdQueue {
					if entry.typ == cmdSend && entry.haveSent {
						entry.errCh <- nil
						deleteIndex = append(deleteIndex, i)
					}
				}
				tcb.cmdQueue = removeCmd(tcb.cmdQueue, deleteIndex)

				// Note that SND.WND is an offset from SND.UNA, that SND.WL1
				// records the sequence number of the last segment used to update
				// SND.WND, and that SND.WL2 records the acknowledgment number of
				// the last segment used to update SND.WND.  The check here
				// prevents using old segments to update the window.
				if tcb.snd.wl1 < hdr.Seq || (tcb.snd.wl1 == hdr.Seq && tcb.snd.wl2 <= hdr.Ack) {
					tcb.snd.wnd = hdr.Window
					tcb.snd.wl1 = hdr.Seq
					tcb.snd.wl2 = hdr.Ack
				}
			} else if hdr.Ack < tcb.snd.una {
				// If the ACK is a duplicate (SEG.ACK < SND.UNA), it can be ignored.
				return nil
			} else if hdr.Ack > tcb.snd.nxt {
				// ??
				// If the ACK acks something not yet sent (SEG.ACK > SND.NXT) then send an ACK, drop the segment, and return.
				return TxHandlerTCP(tcb.local, tcb.foreign, []byte{}, tcb.snd.nxt, tcb.rcv.nxt, ACK, 0, 0)
			}

			switch tcb.state {
			case TCPpcbStateFINWait1:
				// In addition to the processing for the ESTABLISHED state,
				// if our FIN is now acknowledged then enter FIN-WAIT-2 and continue processing in that state.
				tcb.state = TCPpcbStateFINWait2
				return nil
			case TCPpcbStateFINWait2:
				// if the retransmission queue is empty, the user’s CLOSE can be
				// acknowledged ("ok")
				if len(tcb.retxQueue) == 0 {
					var deleteIndex []int
					for i, entry := range tcb.cmdQueue {
						if entry.typ == cmdClose {
							entry.errCh <- nil
							deleteIndex = append(deleteIndex, i)
						}
					}
					tcb.cmdQueue = removeCmd(tcb.cmdQueue, deleteIndex)
				}
				return nil
			case TCPpcbStateEstablished, TCPpcbStateCloseWait:
				// Do the same processing as for the ESTABLISHED state.
				return nil
			case TCPpcbStateClosing:
				// In addition to the processing for the ESTABLISHED state,
				// if the ACK acknowledges our FIN then enter the TIME-WAIT state, otherwise ignore the segment.
				tcb.state = TCPpcbStateTimeWait
				return nil
			case TCPpcbStateLastACK:
				// The only thing that can arrive in this state is an
				// acknowledgment of our FIN.  If our FIN is now acknowledged,
				// delete the TCB, enter the CLOSED state, and return.
				tcb.state = TCPpcbStateClosed
				return nil
			case TCPpcbStateTimeWait:
				// TODO:
				// The only thing that can arrive in this state is a
				// retransmission of the remote FIN.  Acknowledge it, and restart
				// the 2 MSL timeout.
				return nil
			}
		}

		// sixth, check the URG bit,
		switch tcb.state {
		case TCPpcbStateEstablished, TCPpcbStateFINWait1, TCPpcbStateFINWait2:
			if hdr.Flag&URG > 0 {
				tcb.rcv.up = max(tcb.rcv.up, hdr.Urgent)
				// TODO:
				// signal the user that the remote side has urgent data if the urgent
				// pointer (RCV.UP) is in advance of the data consumed.  If the
				// user has already been signaled (or is still in the "urgent
				// mode") for this continuous sequence of urgent data, do not
				// signal the user again.
			}
		case TCPpcbStateCloseWait, TCPpcbStateClosing, TCPpcbStateLastACK, TCPpcbStateTimeWait:
			// ignore
		}

		// seventh, process the segment text
		switch tcb.state {
		case TCPpcbStateEstablished, TCPpcbStateFINWait1, TCPpcbStateFINWait2:
			// TODO:
			// Once in the ESTABLISHED state, it is possible to deliver segment
			// text to user RECEIVE buffers.  Text from segments can be moved
			// into buffers until either the buffer is full or the segment is
			// empty.  If the segment empties and carries an PUSH flag, then
			// the user is informed, when the buffer is returned, that a PUSH
			// has been received.
			copy(tcb.rxQueue[tcb.rxLen:], data)
			tcb.rxLen += uint16(dataLen)
			tcb.rcv.nxt = hdr.Seq + dataLen
			tcb.rcv.wnd = bufferSize - tcb.rxLen

			// TODO:
			// This acknowledgment should be piggybacked on a segment being
			// transmitted if possible without incurring undue delay.
			return TxHandlerTCP(tcb.local, tcb.foreign, []byte{}, tcb.snd.nxt, tcb.rcv.nxt, ACK, tcb.rcv.wnd, 0)
		case TCPpcbStateCloseWait, TCPpcbStateClosing, TCPpcbStateLastACK, TCPpcbStateTimeWait:
			// ignore
		}

		// eighth, check the FIN bit,
		switch tcb.state {
		case TCPpcbStateClosed, TCPpcbStateListen, TCPpcbStateSYNSent:
			// drop the segment and return.
			return nil
		default:
		}

		if hdr.Flag&FIN > 0 {
			// FIN bit is set
			// TODO:
			// signal the user "connection closing" and return any pending RECEIVEs with same message,
			// there is no way to notify a user
			switch tcb.state {
			case TCPpcbStateSYNReceived, TCPpcbStateEstablished:
				tcb.state = TCPpcbStateCloseWait
				return TxHandlerTCP(tcb.local, tcb.foreign, []byte{}, tcb.snd.nxt, tcb.rcv.nxt, ACK, 0, 0)
			case TCPpcbStateFINWait1:
				tcb.state = TCPpcbStateTimeWait
				// TODO:
				// time-wait timer, turn off the other timers; otherwise enter the CLOSING state.
				return TxHandlerTCP(tcb.local, tcb.foreign, []byte{}, tcb.snd.nxt, tcb.rcv.nxt, ACK, 0, 0)
			case TCPpcbStateFINWait2:
				tcb.state = TCPpcbStateTimeWait
				// TODO:
				// time-wait timer, turn off the other timers
				return TxHandlerTCP(tcb.local, tcb.foreign, []byte{}, tcb.snd.nxt, tcb.rcv.nxt, ACK, 0, 0)
			case TCPpcbStateCloseWait, TCPpcbStateClosing, TCPpcbStateLastACK:
				// remain
				return TxHandlerTCP(tcb.local, tcb.foreign, []byte{}, tcb.snd.nxt, tcb.rcv.nxt, ACK, 0, 0)
			case TCPpcbStateTimeWait:
				// TODO:
				// Restart the 2 MSL time-wait timeout.
				return TxHandlerTCP(tcb.local, tcb.foreign, []byte{}, tcb.snd.nxt, tcb.rcv.nxt, ACK, 0, 0)
			}
		}
	}
	return nil
}

func TxHandlerTCP(src TCPEndpoint, dst TCPEndpoint, data []byte, seq uint32, ack uint32, flag ControlFlag, wnd uint16, up uint16) error {

	if len(data)+TCPHeaderSizeMin > IPPayloadSizeMax {
		return fmt.Errorf("data size is too large for TCP payload")
	}

	// transform TCP header to byte strings
	hdr := TCPHeader{
		Src:    src.Port,
		Dst:    dst.Port,
		Seq:    seq,
		Ack:    ack,
		Offset: (TCPHeaderSizeMin >> 2) << 4,
		Flag:   flag,
		Window: wnd,
		Urgent: up,
	}
	data, err := header2dataTCP(&hdr, data, src.Address, dst.Address)
	if err != nil {
		return err
	}

	log.Printf("[D] TCP TxHandler: src=%s,dst=%s,tcp header=%s", src, dst, hdr)
	return TxHandlerIP(IPProtocolTCP, data, src.Address, dst.Address)
}

/*
	TCP timer
*/

const (
	// lower bound and upper bound of RTO
	lbound time.Duration = time.Second      // 1s
	ubound time.Duration = 60 * time.Second // 60s
)

var (
	// smoothed round trip time
	srtt time.Duration = 10 * time.Second

	// the retransmission timeout
	rto time.Duration = 10 * time.Second
)

func tcpTimer(done chan struct{}) {

	for {

		// check if process finishes or not
		select {
		case <-done:
			return
		default:
		}

		time.Sleep(time.Second)
		tcpMutex.Lock()
		for _, tcb := range tcbs {
			tcb.retxQueue = removeQueue(tcb.retxQueue, tcb.snd.una)
			var deleteIndex []int
			for i, entry := range tcb.retxQueue {
				// retransmittion timeout
				if entry.txTime.Add(rto).Before(time.Now()) {
					entry.retxCount++
					if entry.retxCount >= maxRetxCount {
						// TODO:
						// notify user that retransmit time is over.
						deleteIndex = append(deleteIndex, i)
						log.Printf("[E] retransmission count is over, network may be not connected")
					}
					err := TxHandlerTCP(tcb.local, tcb.foreign, entry.data, entry.seq, 0, entry.flag, tcb.snd.wnd, 0)
					if err != nil {
						log.Printf("[E] : retransmit error %s", err)
					}
				}
			}
			tcb.retxQueue = removeRetx(tcb.retxQueue, deleteIndex)
		}
		tcpMutex.Unlock()
	}
}
