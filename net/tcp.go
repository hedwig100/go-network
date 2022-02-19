package net

import (
	"fmt"
	"log"
	"math/rand"
	"time"
)

func TCPInit(done chan struct{}) error {
	go tcpTimer(done)
	rand.Seed(time.Now().UnixNano())
	return IPProtocolRegister(&TCPProtocol{})
}

type segment struct {
	seq uint32
	ack uint32
	len uint32
	wnd uint16
	up  uint16
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

func (p *TCPProtocol) rxHandler(data []byte, src IPAddr, dst IPAddr, ipIface *IPIface) error {

	hdr, payload, err := data2headerTCP(data, src, dst)
	// TODO:
	// have to treat header part of payload (payload may have header because we cut data with TCPHeaderSizeMin)
	if err != nil {
		return err
	}

	// search TCP pcb
	var tcb *TCPpcb
	for _, candidate := range tcbs {
		if candidate.local.Address == dst && candidate.local.Port == hdr.Dst {
			tcb = candidate
			break
		}
	}
	if tcb == nil {
		return fmt.Errorf("TCP socket whose address is %s:%d not found", dst, hdr.Dst)
	}

	hdrLen := (hdr.Offset >> 4) << 2
	dataLen := uint32(len(data)) - uint32(hdrLen)
	log.Printf("[D] TCP rxHandler: src=%s:%d,dst=%s:%d,iface=%s,len=%d,tcp header=%s,payload=%v", src, hdr.Src, dst, hdr.Dst, ipIface.Family(), dataLen, hdr, payload)

	// segment
	seg := segment{
		seq: hdr.Seq,
		ack: hdr.Ack,
		len: dataLen,
		wnd: hdr.Window,
		up:  hdr.Urgent,
	}
	if isSet(hdr.Flag, SYN|FIN) {
		seg.len++
	}

	foreign := TCPEndpoint{
		Address: src,
		Port:    hdr.Src,
	}

	return segmentArrives(tcb, seg, hdr.Flag, payload[hdrLen-TCPHeaderSizeMin:], dataLen, foreign)
}

func segmentArrives(tcb *TCPpcb, seg segment, flag ControlFlag, data []byte, dataLen uint32, foreign TCPEndpoint) error {
	tcpMutex.Lock()
	defer tcpMutex.Unlock()

	switch tcb.state {
	case TCPpcbStateClosed:
		if isSet(flag, RST) {
			// An incoming segment containing a RST is discarded
			return nil
		}
		// ACK bit is off
		if !isSet(flag, ACK) {
			return TxHandlerTCP(tcb.local, tcb.foreign, []byte{}, 0, seg.seq+seg.len, RST|ACK, 0, 0)
		}
		// ACK bit is on
		return TxHandlerTCP(tcb.local, tcb.foreign, []byte{}, seg.ack, 0, RST, 0, 0)

	case TCPpcbStateListen:
		// first check for an RST
		if isSet(flag, RST) {
			// An incoming RST should be ignored
			return nil
		}

		// second check for an ACK
		if isSet(flag, ACK) {
			// Any acknowledgment is bad if it arrives on a connection still in the LISTEN state.
			// An acceptable reset segment should be formed for any arriving ACK-bearing segment.
			return TxHandlerTCP(tcb.local, tcb.foreign, []byte{}, seg.ack, 0, RST, 0, 0)
		}

		// third check for a SYN
		if isSet(flag, SYN) {
			// ignore security check

			tcb.foreign = foreign
			tcb.rcv.wnd = bufferSize
			tcb.rcv.nxt = seg.seq + 1
			tcb.irs = seg.seq

			tcb.iss = createISS()
			tcb.snd.nxt = tcb.iss + 1
			tcb.snd.una = tcb.iss
			tcb.foreign = foreign
			tcb.transition(TCPpcbStateSYNReceived)

			copy(tcb.rxQueue[tcb.rxLen:], data)
			tcb.rxLen += uint16(dataLen)
			return TxHelperTCP(tcb, SYN|ACK, []byte{}, 0, nil)
		}

		// fourth other text or control
		log.Printf("[D] TCP segment discarded")
		return nil

	case TCPpcbStateSYNSent:
		// first check the ACK bit
		var acceptable bool
		if isSet(flag, ACK) {
			// If SEG.ACK =< ISS, or SEG.ACK > SND.NXT, send a reset (unless
			// the RST bit is set, if so drop the segment and return)
			if seg.ack <= tcb.iss || seg.ack > tcb.snd.nxt {
				return TxHandlerTCP(tcb.local, tcb.foreign, []byte{}, seg.ack, 0, RST, 0, 0)
			}
			if tcb.snd.una <= seg.ack && seg.ack <= tcb.snd.nxt {
				// this ACK is  acceptable
				acceptable = true
			}
		}

		// second check the RST bit
		if isSet(flag, RST) {
			if acceptable {
				tcb.signalErr("connection reset")
				tcb.transition(TCPpcbStateClosed)
				return nil
			}
			log.Printf("[D] TCP segment discarded")
			return nil
		}

		// third check the security and precedence
		// ignore

		// fourth check the SYN bit
		if isSet(flag, SYN) {
			tcb.rcv.nxt = seg.seq + 1
			tcb.irs = seg.seq

			if acceptable { // our SYN has been ACKed
				tcb.snd.una = seg.ack
				tcb.queueAck()
			}

			if tcb.snd.una > tcb.iss {
				tcb.transition(TCPpcbStateEstablished)
				tcb.snd.wnd = seg.wnd
				tcb.snd.wl1 = seg.seq
				tcb.snd.wl2 = seg.ack

				// notify user call OPEN
				tcb.signalCmd(triggerOpen)
				return TxHelperTCP(tcb, ACK, []byte{}, 0, nil)
			} else {
				tcb.transition(TCPpcbStateSYNReceived)
				// TODO:
				// If there are other controls or text in the
				// segment, queue them for processing after the ESTABLISHED state
				// has been reached
				return TxHelperTCP(tcb, SYN|ACK, []byte{}, 0, nil)
			}
		}

		// fifth, if neither of the SYN or RST bits is set then drop the segment and return.
		log.Printf("[D] TCP segment discarded")
		return nil

	default:
		// first check sequence number
		var acceptable bool
		if tcb.rcv.wnd == 0 {
			if seg.len == 0 && seg.seq == tcb.rcv.nxt {
				acceptable = true
			}
			if seg.len > 0 {
				acceptable = false
			}
		} else {
			if seg.len == 0 && (tcb.rcv.nxt <= seg.seq && seg.seq < tcb.rcv.nxt+uint32(tcb.rcv.wnd)) {
				acceptable = true
			}
			if seg.len > 0 && (tcb.rcv.nxt <= seg.seq && seg.seq < tcb.rcv.nxt+uint32(tcb.rcv.wnd)) || (tcb.rcv.nxt <= seg.seq+seg.len-1 && seg.seq+seg.len-1 < tcb.rcv.nxt+uint32(tcb.rcv.wnd)) {
				acceptable = true
			}
		}
		if !acceptable {
			if isSet(flag, RST) {
				return nil
			}
			log.Printf("[E] ACK number is not acceptable")
			return TxHelperTCP(tcb, ACK, []byte{}, 0, nil)
		}

		// TODO:
		// In the following it is assumed that the segment is the idealized
		// segment that begins at RCV.NXT and does not exceed the window.
		// One could tailor actual segments to fit this assumption by
		// trimming off any portions that lie outside the window (including
		// SYN and FIN), and only processing further if the segment then
		// begins at RCV.NXT.  Segments with higher begining sequence
		// numbers may be held for later processing.
		//
		// right := min(tcb.rcv.nxt+uint32(tcb.rcv.wnd)-seg.seq, dataLen)
		// data = data[tcb.rcv.nxt-seg.seq : right]

		// second check the RST bit
		switch tcb.state {
		case TCPpcbStateSYNReceived:
			if isSet(flag, RST) {
				// If this connection was initiated with an active OPEN (i.e., came
				// from SYN-SENT state) then the connection was refused, signal
				// the user "connection refused".
				tcb.signalErr("connection refused")
				tcb.transition(TCPpcbStateClosed)
				return nil
			}
		case TCPpcbStateEstablished, TCPpcbStateFINWait1, TCPpcbStateFINWait2, TCPpcbStateCloseWait:
			if isSet(flag, RST) {
				// any outstanding RECEIVEs and SEND should receive "reset" responses
				// Users should also receive an unsolicited general "connection reset" signal
				tcb.signalErr("connection reset")
				tcb.transition(TCPpcbStateClosed)
				return nil
			}
		case TCPpcbStateClosing, TCPpcbStateLastACK, TCPpcbStateTimeWait:
			if isSet(flag, RST) {
				tcb.signalErr("connection reset")
				tcb.transition(TCPpcbStateClosed)
				return nil
			}
		}

		// third check security and precedence
		// ignore

		// fourth, check the SYN bit
		switch tcb.state {
		case TCPpcbStateSYNReceived, TCPpcbStateEstablished, TCPpcbStateFINWait1, TCPpcbStateFINWait2,
			TCPpcbStateCloseWait, TCPpcbStateClosing, TCPpcbStateLastACK, TCPpcbStateTimeWait:
			if isSet(flag, SYN) {
				// any outstanding RECEIVEs and SEND should receive "reset" responses,
				// all segment queues should be flushed, the user should also
				// receive an unsolicited general "connection reset" signal
				tcb.signalErr("connection reset")
				tcb.transition(TCPpcbStateClosed)
				return TxHelperTCP(tcb, RST, []byte{}, 0, nil)
			}
		}

		// fifth check the ACK field
		if !isSet(flag, ACK) {
			log.Printf("[D] TCP segment discarded")
			return nil
		}
		switch tcb.state {
		case TCPpcbStateSYNReceived:
			if tcb.snd.una <= seg.ack && seg.ack <= tcb.snd.nxt {
				tcb.transition(TCPpcbStateEstablished)
			} else {
				log.Printf("unacceptable ACK is sent")
				return TxHandlerTCP(tcb.local, tcb.foreign, []byte{}, seg.ack, 0, RST, 0, 0)
			}
			fallthrough
		case TCPpcbStateEstablished, TCPpcbStateFINWait1, TCPpcbStateFINWait2, TCPpcbStateCloseWait, TCPpcbStateClosing:
			if tcb.snd.una < seg.ack && seg.ack <= tcb.snd.nxt {
				tcb.snd.una = seg.ack

				// Users should receive
				// positive acknowledgments for buffers which have been SENT and
				// fully acknowledged (i.e., SEND buffer should be returned with
				// "ok" response)
				// in removeQueue function
				tcb.queueAck()

				// Note that SND.WND is an offset from SND.UNA, that SND.WL1
				// records the sequence number of the last segment used to update
				// SND.WND, and that SND.WL2 records the acknowledgment number of
				// the last segment used to update SND.WND.  The check here
				// prevents using old segments to update the window.
				if tcb.snd.wl1 < seg.seq || (tcb.snd.wl1 == seg.seq && tcb.snd.wl2 <= seg.ack) {
					tcb.snd.wnd = seg.wnd
					tcb.snd.wl1 = seg.seq
					tcb.snd.wl2 = seg.ack
				}
			} else if seg.ack < tcb.snd.una {
				// If the ACK is a duplicate (SEG.ACK < SND.UNA), it can be ignored.
			} else if seg.ack > tcb.snd.nxt {
				// ??
				// If the ACK acks something not yet sent (SEG.ACK > SND.NXT) then send an ACK, drop the segment, and return.
				return TxHelperTCP(tcb, ACK, []byte{}, 0, nil)
			}

			switch tcb.state {
			case TCPpcbStateFINWait1:
				// In addition to the processing for the ESTABLISHED state,
				// if our FIN is now acknowledged then enter FIN-WAIT-2 and continue processing in that state.
				if seg.ack == tcb.snd.nxt {
					tcb.transition(TCPpcbStateFINWait2)
				}
			case TCPpcbStateFINWait2:
				// if the retransmission queue is empty, the user’s CLOSE can be
				// acknowledged ("ok")
				if len(tcb.retxQueue) == 0 {
					tcb.signalCmd(triggerClose)
				}
			case TCPpcbStateEstablished, TCPpcbStateCloseWait:
				// Do the same processing as for the ESTABLISHED state.
			case TCPpcbStateClosing:
				// In addition to the processing for the ESTABLISHED state,
				// if the ACK acknowledges our FIN then enter the TIME-WAIT state, otherwise ignore the segment.
				if seg.ack == tcb.snd.nxt {
					tcb.transition(TCPpcbStateTimeWait)
					tcb.lastTxTime = time.Now()
				}
			}

		case TCPpcbStateLastACK:
			// The only thing that can arrive in this state is an
			// acknowledgment of our FIN.  If our FIN is now acknowledged,
			// delete the TCB, enter the CLOSED state, and return.
			if seg.ack == tcb.snd.nxt {
				tcb.signalErr("connection closed")
				tcb.transition(TCPpcbStateClosed)
			}
			return nil
		case TCPpcbStateTimeWait:
			// The only thing that can arrive in this state is a
			// retransmission of the remote FIN.  Acknowledge it, and restart
			// the 2 MSL timeout.
			if isSet(flag, FIN) {
				tcb.lastTxTime = time.Now()
			}
			return nil
		}

		// sixth, check the URG bit,
		switch tcb.state {
		case TCPpcbStateEstablished, TCPpcbStateFINWait1, TCPpcbStateFINWait2:
			if isSet(flag, RST) {
				tcb.rcv.up = max(tcb.rcv.up, seg.up)
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
			// Once in the ESTABLISHED state, it is possible to deliver segment
			// text to user RECEIVE buffers.  Text from segments can be moved
			// into buffers until either the buffer is full or the segment is
			// empty.  If the segment empties and carries an PUSH flag, then
			// the user is informed, when the buffer is returned, that a PUSH
			// has been received.
			if dataLen > 0 {
				copy(tcb.rxQueue[tcb.rxLen:], data)
				tcb.rxLen += uint16(dataLen)
				tcb.rcv.nxt = seg.seq + seg.len
				tcb.rcv.wnd -= uint16(dataLen)
				tcb.signalCmd(triggerReceive)

				// TODO:
				// This acknowledgment should be piggybacked on a segment being
				// transmitted if possible without incurring undue delay.
				return TxHelperTCP(tcb, ACK, []byte{}, 0, nil)
			}
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

		if isSet(flag, FIN) {
			// FIN bit is set
			// signal the user "connection closing" and return any pending RECEIVEs with same message,
			tcb.signalCmd(triggerReceive)
			tcb.rcv.nxt = seg.seq + 1

			switch tcb.state {
			case TCPpcbStateSYNReceived, TCPpcbStateEstablished:
				tcb.transition(TCPpcbStateCloseWait)
				return TxHelperTCP(tcb, ACK, []byte{}, 0, nil)
			case TCPpcbStateFINWait1:
				if seg.ack == tcb.snd.nxt {
					tcb.transition(TCPpcbStateTimeWait)
					// time-wait timer, turn off the other timers
					tcb.lastTxTime = time.Now()
				} else {
					tcb.transition(TCPpcbStateClosing)
				}
				return TxHelperTCP(tcb, ACK, []byte{}, 0, nil)
			case TCPpcbStateFINWait2:
				tcb.transition(TCPpcbStateTimeWait)
				// time-wait timer, turn off the other timers
				tcb.lastTxTime = time.Now()
				return TxHelperTCP(tcb, ACK, []byte{}, 0, nil)
			case TCPpcbStateCloseWait, TCPpcbStateClosing, TCPpcbStateLastACK:
				// remain
				return TxHelperTCP(tcb, ACK, []byte{}, 0, nil)
			case TCPpcbStateTimeWait:
				// Restart the 2 MSL time-wait timeout.
				tcb.lastTxTime = time.Now()
				return TxHelperTCP(tcb, ACK, []byte{}, 0, nil)
			}
		}
	}
	return nil
}

func TxHelperTCP(tcb *TCPpcb, flag ControlFlag, data []byte, trigger uint8, errCh chan error) error {
	seq := tcb.snd.nxt
	if isSet(flag, SYN) {
		seq = tcb.iss
	}
	if err := TxHandlerTCP(tcb.local, tcb.foreign, data, seq, tcb.rcv.nxt, flag, tcb.rcv.wnd, tcb.rcv.up); err != nil {
		return err
	}
	if isSet(flag, SYN|FIN) || len(data) > 0 {
		tcb.queueAdd(seq, flag, data, trigger, errCh)
	}
	return nil
}

func TxHandlerTCP(src TCPEndpoint, dst TCPEndpoint, payload []byte, seq uint32, ack uint32, flag ControlFlag, wnd uint16, up uint16) error {

	if len(payload)+TCPHeaderSizeMin > IPPayloadSizeMax {
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
	data, err := header2dataTCP(&hdr, payload, src.Address, dst.Address)
	if err != nil {
		return err
	}

	log.Printf("[D] TCP TxHandler: src=%s,dst=%s,len=%d,tcp header=%s", src, dst, len(payload), hdr)
	return TxHandlerIP(IPProtocolTCP, data, src.Address, dst.Address)
}
