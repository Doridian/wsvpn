package servers

import (
	"errors"
	"log"

	"github.com/songgao/water"
)

func (s *Server) serveTAP() {
	defer panic(errors.New("TAP closed"))

	packet := make([]byte, s.packetBufferSize)

	for {
		n, err := s.tapIface.Read(packet)
		if err != nil {
			log.Printf("[%s] Error reading packet from TAP: %v", s.ServerID, err)
			return
		}

		_, err = s.SocketGroup.HandlePacket(nil, packet[:n])
		if err != nil {
			log.Printf("[%s] Error handling packet from TAP: %v", s.ServerID, err)
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
	err = s.extendTAPConfig(&tapConfig)
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
