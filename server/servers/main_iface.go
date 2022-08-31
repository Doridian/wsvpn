package servers

import (
	"errors"

	"github.com/Doridian/water"
	"github.com/Doridian/wsvpn/shared/iface"
)

func (s *Server) serveMainIface() {
	defer func() {
		s.setServeError(errors.New("main iface closed"))
		s.serveWaitGroup.Done()
	}()

	packet := make([]byte, 0)

	for {
		if len(packet) != s.packetBufferSize {
			packet = make([]byte, s.packetBufferSize)
		}

		n, err := s.mainIface.Interface.Read(packet)
		if err != nil {
			s.log.Printf("Error reading packet from main iface: %v", err)
			return
		}

		if n < 1 || n >= len(packet) {
			continue
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
	err = iface.GetPlatformSpecifics(&ifaceConfig, s.InterfaceConfig)
	if err != nil {
		return err
	}

	mainIface, err := water.New(ifaceConfig)
	if err != nil {
		return err
	}
	s.mainIface = iface.NewInterfaceWrapper(mainIface)

	s.ifaceCreationMutex.Unlock()

	vpnNet := s.VPNNet
	if s.InterfaceConfig.OneInterfacePerConnection {
		vpnNet = nil
	}

	serverIp := vpnNet.GetServerIP()

	err = s.mainIface.Configure(serverIp, vpnNet, serverIp)
	if err != nil {
		return err
	}
	return s.mainIface.SetMTU(s.mtu)
}
