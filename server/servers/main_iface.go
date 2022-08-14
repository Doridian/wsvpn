package servers

import (
	"errors"

	"github.com/songgao/water"
)

func (s *Server) serveMainIface() {
	defer func() {
		s.setServeError(errors.New("main iface closed"))
		s.serveWaitGroup.Done()
	}()

	// TODO: For change-able MTU we need to re-create this buffer
	packet := make([]byte, s.packetBufferSize)

	for {
		n, err := s.mainIface.Read(packet)
		if err != nil {
			s.log.Printf("Error reading packet from main iface: %v", err)
			return
		}

		_, err = s.PacketHandler.HandlePacket(nil, packet[:n])
		if err != nil {
			s.log.Printf("Error handling packet from main iface: %v", err)
			return
		}
	}
}

func (s *Server) createMainIface() error {
	var err error

	s.ifaceCreationMutex.Lock()
	ifaceConfig := water.Config{
		DeviceType: s.Mode.ToWaterDeviceType(),
	}
	err = s.getPlatformSpecifics(&ifaceConfig)
	if err != nil {
		return err
	}

	s.mainIface, err = water.New(ifaceConfig)
	if err != nil {
		return err
	}
	s.ifaceCreationMutex.Unlock()

	return s.configIface(s.mainIface, s.VPNNet.GetServerIP())
}
