package sockets

import (
	"time"
)

func (s *Socket) installPingPongHandlers() {
	if s.pingInterval <= 0 || s.pingTimeout <= 0 {
		s.log.Println("Ping disabled")
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
			case <-time.After(s.pingInterval):
				pingTimeoutTimer.Stop()
				err := s.adapter.WritePingMessage()
				if err != nil {
					s.log.Printf("Error sending ping: %v", err)
					return
				}
				pingTimeoutTimer.Reset(s.pingTimeout)
			case <-pingTimeoutTimer.C:
				s.log.Println("Ping timeout")
				return
			case <-s.closechan:
				return
			}
		}
	}()

	s.log.Printf("Ping enabled with interval %v and timeout %v", s.pingInterval, s.pingTimeout)
}
