package servers

import (
	"errors"

	"github.com/songgao/water"
)

func (s *Server) serveTAP() {
	defer func() {
		s.serveErrorChannel <- errors.New("TAP closed")
		s.serveWaitGroup.Done()
	}()

	// TODO: For change-able MTU we need to re-create this buffer
	packet := make([]byte, s.packetBufferSize)

	for {
		n, err := s.tapIface.Read(packet)
		if err != nil {
			s.log.Printf("Error reading packet from TAP: %v", err)
			return
		}

		_, err = s.PacketHandler.HandlePacket(nil, packet[:n])
		if err != nil {
			s.log.Printf("Error handling packet from TAP: %v", err)
			return
		}
	}
}

func (s *Server) createTAP() error {
	var err error

	s.ifaceCreationMutex.Lock()
	tapConfig := water.Config{
		DeviceType: water.TAP,
	}
	err = s.getPlatformSpecifics(&tapConfig, s.InterfacesConfig)
	if err != nil {
		return err
	}

	s.tapIface, err = water.New(tapConfig)
	if err != nil {
		return err
	}
	s.ifaceCreationMutex.Unlock()

	return s.configIface(s.tapIface, s.VPNNet.GetServerIP())
}
