//go:build linux

package servers

import (
	"fmt"
	"net"

	"github.com/Doridian/water"
	"github.com/Doridian/wsvpn/shared"
)

func (s *Server) configureInterfaceMTU(dev *water.Interface) error {
	return shared.ExecCmd("ip", "link", "set", "dev", dev.Name(), "mtu", fmt.Sprintf("%d", s.mtu))
}

func (s *Server) configIface(dev *water.Interface, ipClient net.IP) error {
	err := s.configureInterfaceMTU(dev)
	if err != nil {
		return err
	}

	err = shared.ExecCmd("ip", "link", "set", "dev", dev.Name(), "up")
	if err != nil {
		return err
	}

	if !s.DoLocalIpConfig {
		return nil
	}

	if s.InterfaceConfig.OneInterfacePerConnection {
		return shared.ExecCmd("ip", "addr", "add", "dev", dev.Name(), s.VPNNet.GetServerIP().String(), "peer", ipClient.String())
	}
	return shared.ExecCmd("ip", "addr", "add", "dev", dev.Name(), fmt.Sprintf("%s/%d", s.VPNNet.GetServerIP().String(), s.VPNNet.GetSize()))
}

func (s *Server) getPlatformSpecifics(config *water.Config) error {
	if s.InterfaceConfig.OneInterfacePerConnection {
		if s.InterfaceConfig.Name != "" {
			config.Name = shared.FindLowestNetworkInterfaceByPrefix(s.InterfaceConfig.Name)
		}
		config.Persist = false

		return nil
	}

	config.Name = s.InterfaceConfig.Name
	config.Persist = s.InterfaceConfig.Persist

	return nil
}

func (s *Server) verifyPlatformFlags() error {
	return nil
}
