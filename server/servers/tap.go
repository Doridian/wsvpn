package servers

import (
	"errors"

	"github.com/songgao/water"
)

func (s *Server) serveTAP() {
	defer panic(errors.New("TAP closed")) // TODO: This should not panic

	packet := make([]byte, s.packetBufferSize)

	for {
		n, err := s.tapIface.Read(packet)
		if err != nil {
			s.log.Printf("Error reading packet from TAP: %v", err)
			return
		}

		_, err = s.SocketGroup.HandlePacket(nil, packet[:n])
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
