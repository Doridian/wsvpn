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
			log.Printf("[S] Error reading packet from TAP: %v", err)
			return
		}

		_, err = s.SocketGroup.HandlePacket(nil, packet[:n])
		if err != nil {
			log.Printf("[S] Error handling packet from TAP: %v", err)
			return
		}
	}
}

func (s *Server) createTAP() error {
	s.ifaceCreationMutex.Lock()
	tapConfig := water.Config{
		DeviceType: water.TAP,
	}
	err := s.extendTAPConfig(&tapConfig)
	if err != nil {
		return err
	}

	tapDev, err := water.New(tapConfig)
	if err != nil {
		return err
	}
	s.ifaceCreationMutex.Unlock()

	return s.configIface(tapDev, s.VPNNet.GetServerIP())
}
