package net

import (
	"fmt"
	"log"
	"time"
)

/*
	TCP timer
*/

const (
	// lower bound and upper bound of RTO
	lbound time.Duration = time.Second      // 1s
	ubound time.Duration = 60 * time.Second // 60s

	MSL time.Duration = 2 * time.Minute
)

var (
	// smoothed round trip time
	srtt time.Duration = 10 * time.Second

	// the retransmission timeout
	rto time.Duration = 10 * time.Second
)

func calculateRTO(rtt time.Duration) {
	// ALPHA = 0.7
	// BETA = 1.7
	srtt = 7*srtt/10 + 3*rtt/10
	if lbound > 17*srtt/10 {
		rto = lbound
	} else if ubound < 17*srtt/10 {
		rto = ubound
	} else {
		rto = 17 * srtt / 10
	}
	log.Printf("[I] RTT=%s,RTO=%s", rtt, rto)
}

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

			// time-wait timeout
			if tcb.state == TCPpcbStateTimeWait && tcb.lastTxTime.Add(MSL).Before(time.Now()) {
				tcb.signalErr("connection aborted due to user timeout")
				tcb.transition(TCPpcbStateClosed)
				continue
			}

			tcb.queueAck()
			var deleteIndex []int
			for i, entry := range tcb.retxQueue {

				// user timeout
				if entry.first.Add(tcb.timeout).Before(time.Now()) {
					tcb.signalErr("connection aborted due to user timeout")
					tcb.transition(TCPpcbStateClosed)
					break
				}

				// retransmission
				rtoNow := rto * (1 << entry.retxCount)
				if entry.last.Add(rtoNow).Before(time.Now()) {
					entry.retxCount++

					if entry.retxCount >= maxRetxCount { // retransmission time is over than limit
						// notify user
						if entry.errCh != nil {
							entry.errCh <- fmt.Errorf("retransmission time is over than limit,network may be not connected")
						}
						deleteIndex = append(deleteIndex, i)
					} else { // retransmission
						log.Printf("[I] restransmission time=%d,local=%s,foreign=%s,seq=%d,flag=%s", entry.retxCount, tcb.local, tcb.foreign, entry.seq, entry.flag)
						err := TxHandlerTCP(tcb.local, tcb.foreign, entry.data, entry.seq, tcb.rcv.nxt, entry.flag, tcb.snd.wnd, 0)
						if err != nil {
							log.Printf("[E] : retransmit error %s", err)
						}
						entry.last = time.Now()
					}
				}
			}
			tcb.retxQueue = removeRetx(tcb.retxQueue, deleteIndex)
		}

		tcpMutex.Unlock()
	}
}
