package tcp

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
		mutex.Lock()

		for _, pcb := range pcbs {

			// time-wait timeout
			if pcb.state == PCBStateTimeWait && pcb.lastTxTime.Add(MSL).Before(time.Now()) {
				pcb.signalErr("connection aborted due to user timeout")
				pcb.transition(PCBStateClosed)
				continue
			}

			pcb.queueAck()
			var deleteIndex []int
			for i, entry := range pcb.retxQueue {

				// user timeout
				if entry.first.Add(pcb.timeout).Before(time.Now()) {
					pcb.signalErr("connection aborted due to user timeout")
					pcb.transition(PCBStateClosed)
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
						log.Printf("[I] restransmission time=%d,local=%s,foreign=%s,seq=%d,flag=%s", entry.retxCount, pcb.local, pcb.foreign, entry.seq, entry.flag)
						err := TxHandlerTCP(pcb.local, pcb.foreign, entry.data, entry.seq, pcb.rcv.nxt, entry.flag, pcb.snd.wnd, 0)
						if err != nil {
							log.Printf("[E] : retransmit error %s", err)
						}
						entry.last = time.Now()
					}
				}
			}
			pcb.retxQueue = removeRetx(pcb.retxQueue, deleteIndex)
		}

		mutex.Unlock()
	}
}
