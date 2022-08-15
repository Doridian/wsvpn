//go:build darwin

package servers

import (
	"fmt"
	"log"
	"net"

	"github.com/Doridian/wsvpn/shared"
	"github.com/songgao/water"
)

func (s *Server) configureInterfaceMTU(dev *water.Interface) error {
	return shared.ExecCmd("ifconfig", dev.Name(), "mtu", fmt.Sprintf("%d", s.mtu))
}

func (s *Server) configIface(dev *water.Interface, ipClient net.IP) error {
	err := s.configureInterfaceMTU(dev)
	if err != nil {
		return err
	}

	if !s.DoLocalIpConfig {
		return shared.ExecCmd("ifconfig", dev.Name(), "up")
	}

	if s.InterfaceConfig.OneInterfacePerConnection {
		return shared.ExecCmd("ifconfig", dev.Name(), s.VPNNet.GetServerIP().String(), ipClient.String(), "up")
	}

	err = shared.ExecCmd("ifconfig", dev.Name(), fmt.Sprintf("%s/%d", s.VPNNet.GetServerIP().String(), s.VPNNet.GetSize()), "up")
	if err != nil {
		err = shared.ExecCmd("ifconfig", dev.Name(), fmt.Sprintf("%s/%d", s.VPNNet.GetServerIP().String(), s.VPNNet.GetSize()), ipClient.String(), "up")
	}

	if err != nil {
		return err
	}

	err = shared.ExecCmd("route", "add", "-net", fmt.Sprintf("%s/%d", s.VPNNet.GetServerIP().String(), s.VPNNet.GetSize()), "-interface", dev.Name())
	if err != nil {
		log.Printf("Error adding route: %v", err)
	}

	return nil
}

func (s *Server) getPlatformSpecifics(config *water.Config) error {
	if s.InterfaceConfig.OneInterfacePerConnection {
		if s.InterfaceConfig.Name != "" {
			config.Name = shared.FindLowestNetworkInterfaceByPrefix(s.InterfaceConfig.Name)
		}

		return nil
	}

	config.Name = s.InterfaceConfig.Name

	return nil
}

func (s *Server) verifyPlatformFlags() error {
	return nil
}
