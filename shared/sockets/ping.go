package sockets

import (
	"flag"
	"log"
	"time"
)

var pingIntervalPtr = flag.Duration("ping-interval", time.Second*time.Duration(30), "Send ping frames in this interval")
var pingTimeoutPtr = flag.Duration("ping-timeout", time.Second*time.Duration(5), "Disconnect if no ping response is received after timeout")

func (s *Socket) installPingPongHandlers(pingInterval time.Duration, pingTimeout time.Duration) {
	if pingInterval <= 0 || pingTimeout <= 0 {
		log.Printf("[%s] Ping disabled", s.ConnectionID)
		return
	}

	// Create a dummy timer that won't ever run so we can wait for it
	pingTimeoutTimer := time.NewTimer(time.Hour)
	pingTimeoutTimer.Stop()

	s.adapter.SetPongHandler(func() {
		pingTimeoutTimer.Stop()
	})

	s.wg.Add(1)

	go func() {
		defer s.closeDone()
		defer pingTimeoutTimer.Stop()

		for {
			select {
			case <-time.After(pingInterval):
				pingTimeoutTimer.Stop()
				err := s.adapter.WritePingMessage()
				if err != nil {
					log.Printf("[%s] Error sending ping: %v", s.ConnectionID, err)
					return
				}
				pingTimeoutTimer.Reset(pingTimeout)
			case <-pingTimeoutTimer.C:
				log.Printf("[%s] Ping timeout", s.ConnectionID)
				return
			case <-s.closechan:
				return
			}
		}
	}()

	log.Printf("[%s] Ping enabled with interval %v and timeout %v", s.ConnectionID, pingInterval, pingTimeout)
}
