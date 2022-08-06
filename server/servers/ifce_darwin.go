//go:build darwin

package servers

import (
	"fmt"
	"net"

	"github.com/Doridian/wsvpn/shared"
	"github.com/songgao/water"
)

func (s *Server) configIface(dev *water.Interface, ipClient net.IP) error {
	err := shared.ExecCmd("ifconfig", dev.Name(), "mtu", fmt.Sprintf("%d", s.mtu))
	if err != nil {
		return err
	}

	if !s.DoLocalIpConfig {
		return shared.ExecCmd("ifconfig", dev.Name(), "up")
	}

	if s.InterfaceConfig.OneInterfacePerConnection {
		return shared.ExecCmd("ifconfig", dev.Name(), s.VPNNet.GetServerIP().String(), ipClient.String(), "up")
	}
	return shared.ExecCmd("ifconfig", dev.Name(), fmt.Sprintf("%s/%d", s.VPNNet.GetServerIP().String(), s.VPNNet.GetSize()), "up")
}

func (s *Server) getPlatformSpecifics(config *water.Config) error {
	return nil
}

func (s *Server) verifyPlatformFlags() error {
	return nil
}
