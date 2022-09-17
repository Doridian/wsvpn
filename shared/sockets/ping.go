package sockets

import (
	"time"
)

func (s *Socket) installPingPongHandlers() {
	// Create a dummy timer that won't ever run so we can wait for it
	pingTimeoutTimer := time.NewTimer(time.Hour)
	pingTimeoutTimer.Stop()

	s.adapter.SetPongHandler(func() {
		s.log.Println("Received pong")
		pingTimeoutTimer.Stop()
	})

	s.wg.Add(1)

	go func() {
		defer s.closeDone()
		defer pingTimeoutTimer.Stop()

		for {
			pingEnabled := true
			pingInterval := s.pingInterval
			if s.pingInterval <= 0 || s.pingTimeout <= 0 {
				pingInterval = time.Duration(10 * time.Second)
				pingEnabled = false
			}

			select {
			case <-time.After(pingInterval):
				pingTimeoutTimer.Stop()
				if !pingEnabled {
					continue
				}
				s.log.Println("Sent ping")
				err := s.adapter.WritePingMessage()
				if err != nil {
					s.log.Printf("Error sending ping: %v", err)
					return
				}
				pingTimeoutTimer.Reset(s.pingTimeout)
			case <-pingTimeoutTimer.C:
				s.log.Println("Ping timeout")
				return
			case <-s.closeChan:
				return
			}
		}
	}()
}
