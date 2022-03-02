package tcp

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/hedwig100/go-network/pkg/ip"
	"github.com/hedwig100/go-network/pkg/utils"
)

func Init(done chan struct{}) error {
	go tcpTimer(done)
	rand.Seed(time.Now().UnixNano())
	return ip.ProtoRegister(&Proto{})
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
// Proto is struct for TCP protocol handler.
// This implements IPUpperProtocol interface.
type Proto struct{}

func (p *Proto) Type() ip.ProtoType {
	return ip.ProtoTCP
}

func (p *Proto) RxHandler(data []byte, src ip.Addr, dst ip.Addr, ipIface *ip.Iface) error {

	hdr, payload, err := data2header(data, src, dst)
	// TODO:
	// have to treat header part of payload (payload may have header because we cut data with-HeaderSizeMin)
	if err != nil {
		return err
	}

	// search TCP pcb
	var pcb *pcb
	for _, candidate := range pcbs {
		if candidate.local.Addr == dst && candidate.local.Port == hdr.Dst {
			pcb = candidate
			break
		}
	}
	if pcb == nil {
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

	foreign := Endpoint{
		Addr: src,
		Port: hdr.Src,
	}

	return segmentArrives(pcb, seg, hdr.Flag, payload[hdrLen-HeaderSizeMin:], dataLen, foreign)
}

func segmentArrives(pcb *pcb, seg segment, flag ControlFlag, data []byte, dataLen uint32, foreign Endpoint) error {
	mutex.Lock()
	defer mutex.Unlock()

	switch pcb.state {
	case PCBStateClosed:
		if isSet(flag, RST) {
			// An incoming segment containing a RST is discarded
			return nil
		}
		// ACK bit is off
		if !isSet(flag, ACK) {
			return TxHandler(pcb.local, pcb.foreign, []byte{}, 0, seg.seq+seg.len, RST|ACK, 0, 0)
		}
		// ACK bit is on
		return TxHandler(pcb.local, pcb.foreign, []byte{}, seg.ack, 0, RST, 0, 0)

	case PCBStateListen:
		// first check for an RST
		if isSet(flag, RST) {
			// An incoming RST should be ignored
			return nil
		}

		// second check for an ACK
		if isSet(flag, ACK) {
			// Any acknowledgment is bad if it arrives on a connection still in the LISTEN state.
			// An acceptable reset segment should be formed for any arriving ACK-bearing segment.
			return TxHandler(pcb.local, pcb.foreign, []byte{}, seg.ack, 0, RST, 0, 0)
		}

		// third check for a SYN
		if isSet(flag, SYN) {
			// ignore security check

			pcb.foreign = foreign
			pcb.rcv.wnd = bufferSize
			pcb.rcv.nxt = seg.seq + 1
			pcb.irs = seg.seq

			pcb.iss = createISS()
			pcb.snd.nxt = pcb.iss + 1
			pcb.snd.una = pcb.iss
			pcb.foreign = foreign
			pcb.transition(PCBStateSYNReceived)

			copy(pcb.rxQueue[pcb.rxLen:], data)
			pcb.rxLen += uint16(dataLen)
			return TxHelperTCP(pcb, SYN|ACK, []byte{}, 0, nil)
		}

		// fourth other text or control
		log.Printf("[D] TCP segment discarded")
		return nil

	case PCBStateSYNSent:
		// first check the ACK bit
		var acceptable bool
		if isSet(flag, ACK) {
			// If SEG.ACK =< ISS, or SEG.ACK > SND.NXT, send a reset (unless
			// the RST bit is set, if so drop the segment and return)
			if seg.ack <= pcb.iss || seg.ack > pcb.snd.nxt {
				return TxHandler(pcb.local, pcb.foreign, []byte{}, seg.ack, 0, RST, 0, 0)
			}
			if pcb.snd.una <= seg.ack && seg.ack <= pcb.snd.nxt {
				// this ACK is  acceptable
				acceptable = true
			}
		}

		// second check the RST bit
		if isSet(flag, RST) {
			if acceptable {
				pcb.signalErr("connection reset")
				pcb.transition(PCBStateClosed)
				return nil
			}
			log.Printf("[D] TCP segment discarded")
			return nil
		}

		// third check the security and precedence
		// ignore

		// fourth check the SYN bit
		if isSet(flag, SYN) {
			pcb.rcv.nxt = seg.seq + 1
			pcb.irs = seg.seq

			if acceptable { // our SYN has been ACKed
				pcb.snd.una = seg.ack
				pcb.queueAck()
			}

			if pcb.snd.una > pcb.iss {
				pcb.transition(PCBStateEstablished)
				pcb.snd.wnd = seg.wnd
				pcb.snd.wl1 = seg.seq
				pcb.snd.wl2 = seg.ack

				// notify user call OPEN
				pcb.signalCmd(triggerOpen)
				return TxHelperTCP(pcb, ACK, []byte{}, 0, nil)
			} else {
				pcb.transition(PCBStateSYNReceived)
				// TODO:
				// If there are other controls or text in the
				// segment, queue them for processing after the ESTABLISHED state
				// has been reached
				return TxHelperTCP(pcb, SYN|ACK, []byte{}, 0, nil)
			}
		}

		// fifth, if neither of the SYN or RST bits is set then drop the segment and return.
		log.Printf("[D] TCP segment discarded")
		return nil

	default:
		// first check sequence number
		var acceptable bool
		if pcb.rcv.wnd == 0 {
			if seg.len == 0 && seg.seq == pcb.rcv.nxt {
				acceptable = true
			}
			if seg.len > 0 {
				acceptable = false
			}
		} else {
			if seg.len == 0 && (pcb.rcv.nxt <= seg.seq && seg.seq < pcb.rcv.nxt+uint32(pcb.rcv.wnd)) {
				acceptable = true
			}
			if seg.len > 0 && (pcb.rcv.nxt <= seg.seq && seg.seq < pcb.rcv.nxt+uint32(pcb.rcv.wnd)) || (pcb.rcv.nxt <= seg.seq+seg.len-1 && seg.seq+seg.len-1 < pcb.rcv.nxt+uint32(pcb.rcv.wnd)) {
				acceptable = true
			}
		}
		if !acceptable {
			if isSet(flag, RST) {
				return nil
			}
			log.Printf("[E] ACK number is not acceptable")
			return TxHelperTCP(pcb, ACK, []byte{}, 0, nil)
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
		// right := min(pcb.rcv.nxt+uint32(pcb.rcv.wnd)-seg.seq, dataLen)
		// data = data[pcb.rcv.nxt-seg.seq : right]

		// second check the RST bit
		switch pcb.state {
		case PCBStateSYNReceived:
			if isSet(flag, RST) {
				// If this connection was initiated with an active OPEN (i.e., came
				// from SYN-SENT state) then the connection was refused, signal
				// the user "connection refused".
				pcb.signalErr("connection refused")
				pcb.transition(PCBStateClosed)
				return nil
			}
		case PCBStateEstablished, PCBStateFINWait1, PCBStateFINWait2, PCBStateCloseWait:
			if isSet(flag, RST) {
				// any outstanding RECEIVEs and SEND should receive "reset" responses
				// Users should also receive an unsolicited general "connection reset" signal
				pcb.signalErr("connection reset")
				pcb.transition(PCBStateClosed)
				return nil
			}
		case PCBStateClosing, PCBStateLastACK, PCBStateTimeWait:
			if isSet(flag, RST) {
				pcb.signalErr("connection reset")
				pcb.transition(PCBStateClosed)
				return nil
			}
		}

		// third check security and precedence
		// ignore

		// fourth, check the SYN bit
		switch pcb.state {
		case PCBStateSYNReceived, PCBStateEstablished, PCBStateFINWait1, PCBStateFINWait2,
			PCBStateCloseWait, PCBStateClosing, PCBStateLastACK, PCBStateTimeWait:
			if isSet(flag, SYN) {
				// any outstanding RECEIVEs and SEND should receive "reset" responses,
				// all segment queues should be flushed, the user should also
				// receive an unsolicited general "connection reset" signal
				pcb.signalErr("connection reset")
				pcb.transition(PCBStateClosed)
				return TxHelperTCP(pcb, RST, []byte{}, 0, nil)
			}
		}

		// fifth check the ACK field
		if !isSet(flag, ACK) {
			log.Printf("[D] TCP segment discarded")
			return nil
		}
		switch pcb.state {
		case PCBStateSYNReceived:
			if pcb.snd.una <= seg.ack && seg.ack <= pcb.snd.nxt {
				pcb.transition(PCBStateEstablished)
			} else {
				log.Printf("unacceptable ACK is sent")
				return TxHandler(pcb.local, pcb.foreign, []byte{}, seg.ack, 0, RST, 0, 0)
			}
			fallthrough
		case PCBStateEstablished, PCBStateFINWait1, PCBStateFINWait2, PCBStateCloseWait, PCBStateClosing:
			if pcb.snd.una < seg.ack && seg.ack <= pcb.snd.nxt {
				pcb.snd.una = seg.ack

				// Users should receive
				// positive acknowledgments for buffers which have been SENT and
				// fully acknowledged (i.e., SEND buffer should be returned with
				// "ok" response)
				// in removeQueue function
				pcb.queueAck()

				// Note that SND.WND is an offset from SND.UNA, that SND.WL1
				// records the sequence number of the last segment used to update
				// SND.WND, and that SND.WL2 records the acknowledgment number of
				// the last segment used to update SND.WND.  The check here
				// prevents using old segments to update the window.
				if pcb.snd.wl1 < seg.seq || (pcb.snd.wl1 == seg.seq && pcb.snd.wl2 <= seg.ack) {
					pcb.snd.wnd = seg.wnd
					pcb.snd.wl1 = seg.seq
					pcb.snd.wl2 = seg.ack
				}
			} else if seg.ack < pcb.snd.una {
				// If the ACK is a duplicate (SEG.ACK < SND.UNA), it can be ignored.
			} else if seg.ack > pcb.snd.nxt {
				// ??
				// If the ACK acks something not yet sent (SEG.ACK > SND.NXT) then send an ACK, drop the segment, and return.
				return TxHelperTCP(pcb, ACK, []byte{}, 0, nil)
			}

			switch pcb.state {
			case PCBStateFINWait1:
				// In addition to the processing for the ESTABLISHED state,
				// if our FIN is now acknowledged then enter FIN-WAIT-2 and continue processing in that state.
				if seg.ack == pcb.snd.nxt {
					pcb.transition(PCBStateFINWait2)
				}
			case PCBStateFINWait2:
				// if the retransmission queue is empty, the user’s CLOSE can be
				// acknowledged ("ok")
				if len(pcb.retxQueue) == 0 {
					pcb.signalCmd(triggerClose)
				}
			case PCBStateEstablished, PCBStateCloseWait:
				// Do the same processing as for the ESTABLISHED state.
			case PCBStateClosing:
				// In addition to the processing for the ESTABLISHED state,
				// if the ACK acknowledges our FIN then enter the TIME-WAIT state, otherwise ignore the segment.
				if seg.ack == pcb.snd.nxt {
					pcb.transition(PCBStateTimeWait)
					pcb.lastTxTime = time.Now()
				}
			}

		case PCBStateLastACK:
			// The only thing that can arrive in this state is an
			// acknowledgment of our FIN.  If our FIN is now acknowledged,
			// delete the TCB, enter the CLOSED state, and return.
			if seg.ack == pcb.snd.nxt {
				pcb.signalErr("connection closed")
				pcb.transition(PCBStateClosed)
			}
			return nil
		case PCBStateTimeWait:
			// The only thing that can arrive in this state is a
			// retransmission of the remote FIN.  Acknowledge it, and restart
			// the 2 MSL timeout.
			if isSet(flag, FIN) {
				pcb.lastTxTime = time.Now()
			}
			return nil
		}

		// sixth, check the URG bit,
		switch pcb.state {
		case PCBStateEstablished, PCBStateFINWait1, PCBStateFINWait2:
			if isSet(flag, RST) {
				pcb.rcv.up = utils.Max(pcb.rcv.up, seg.up)
				// TODO:
				// signal the user that the remote side has urgent data if the urgent
				// pointer (RCV.UP) is in advance of the data consumed.  If the
				// user has already been signaled (or is still in the "urgent
				// mode") for this continuous sequence of urgent data, do not
				// signal the user again.
			}
		case PCBStateCloseWait, PCBStateClosing, PCBStateLastACK, PCBStateTimeWait:
			// ignore
		}

		// seventh, process the segment text
		switch pcb.state {
		case PCBStateEstablished, PCBStateFINWait1, PCBStateFINWait2:
			// Once in the ESTABLISHED state, it is possible to deliver segment
			// text to user RECEIVE buffers.  Text from segments can be moved
			// into buffers until either the buffer is full or the segment is
			// empty.  If the segment empties and carries an PUSH flag, then
			// the user is informed, when the buffer is returned, that a PUSH
			// has been received.
			if dataLen > 0 {
				copy(pcb.rxQueue[pcb.rxLen:], data)
				pcb.rxLen += uint16(dataLen)
				pcb.rcv.nxt = seg.seq + seg.len
				pcb.rcv.wnd -= uint16(dataLen)
				pcb.signalCmd(triggerReceive)

				// TODO:
				// This acknowledgment should be piggybacked on a segment being
				// transmitted if possible without incurring undue delay.
				return TxHelperTCP(pcb, ACK, []byte{}, 0, nil)
			}
		case PCBStateCloseWait, PCBStateClosing, PCBStateLastACK, PCBStateTimeWait:
			// ignore
		}

		// eighth, check the FIN bit,
		switch pcb.state {
		case PCBStateClosed, PCBStateListen, PCBStateSYNSent:
			// drop the segment and return.
			return nil
		default:
		}

		if isSet(flag, FIN) {
			// FIN bit is set
			// signal the user "connection closing" and return any pending RECEIVEs with same message,
			pcb.signalCmd(triggerReceive)
			pcb.rcv.nxt = seg.seq + 1

			switch pcb.state {
			case PCBStateSYNReceived, PCBStateEstablished:
				pcb.transition(PCBStateCloseWait)
				return TxHelperTCP(pcb, ACK, []byte{}, 0, nil)
			case PCBStateFINWait1:
				if seg.ack == pcb.snd.nxt {
					pcb.transition(PCBStateTimeWait)
					// time-wait timer, turn off the other timers
					pcb.lastTxTime = time.Now()
				} else {
					pcb.transition(PCBStateClosing)
				}
				return TxHelperTCP(pcb, ACK, []byte{}, 0, nil)
			case PCBStateFINWait2:
				pcb.transition(PCBStateTimeWait)
				// time-wait timer, turn off the other timers
				pcb.lastTxTime = time.Now()
				return TxHelperTCP(pcb, ACK, []byte{}, 0, nil)
			case PCBStateCloseWait, PCBStateClosing, PCBStateLastACK:
				// remain
				return TxHelperTCP(pcb, ACK, []byte{}, 0, nil)
			case PCBStateTimeWait:
				// Restart the 2 MSL time-wait timeout.
				pcb.lastTxTime = time.Now()
				return TxHelperTCP(pcb, ACK, []byte{}, 0, nil)
			}
		}
	}
	return nil
}

func TxHelperTCP(pcb *pcb, flag ControlFlag, data []byte, trigger uint8, errCh chan error) error {
	seq := pcb.snd.nxt
	if isSet(flag, SYN) {
		seq = pcb.iss
	}
	if err := TxHandler(pcb.local, pcb.foreign, data, seq, pcb.rcv.nxt, flag, pcb.rcv.wnd, pcb.rcv.up); err != nil {
		return err
	}
	if isSet(flag, SYN|FIN) || len(data) > 0 {
		pcb.queueAdd(seq, flag, data, trigger, errCh)
	}
	return nil
}

func TxHandler(src Endpoint, dst Endpoint, payload []byte, seq uint32, ack uint32, flag ControlFlag, wnd uint16, up uint16) error {

	if len(payload)-HeaderSizeMin > ip.PayloadSizeMax {
		return fmt.Errorf("data size is too large for TCP payload")
	}

	// transform TCP header to byte strings
	hdr := Header{
		Src:    src.Port,
		Dst:    dst.Port,
		Seq:    seq,
		Ack:    ack,
		Offset: (HeaderSizeMin >> 2) << 4,
		Flag:   flag,
		Window: wnd,
		Urgent: up,
	}
	data, err := header2data(&hdr, payload, src.Addr, dst.Addr)
	if err != nil {
		return err
	}

	log.Printf("[D] TCP TxHandler: src=%s,dst=%s,len=%d,tcp header=%s", src, dst, len(payload), hdr)
	return ip.TxHandler(ip.ProtoTCP, data, src.Addr, dst.Addr)
}
