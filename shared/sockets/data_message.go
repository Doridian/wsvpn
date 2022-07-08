package sockets

import "fmt"

func (s *Socket) registerDataHandler() {
	s.adapter.SetDataMessageHandler(func(message []byte) bool {
		if s.packetHandler != nil {
			res, err := s.packetHandler.HandlePacket(s, message)
			if err != nil {
				s.log.Printf("Error in packet handler: %v", err)
				return false
			}
			if res {
				return true
			}
		}

		if s.iface == nil {
			return true
		}
		s.iface.Write(message)
		return true
	})
}

func (s *Socket) WriteDataMessage(data []byte) error {
	err := s.adapter.WriteDataMessage(data)
	if err != nil {
		s.CloseError(fmt.Errorf("error sending data message: %v", err))
	}
	return err
}
