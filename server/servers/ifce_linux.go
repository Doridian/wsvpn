//go:build linux

package servers

import (
	"fmt"
	"net"

	"github.com/Doridian/wsvpn/shared"
	"github.com/songgao/water"
)

func (s *Server) configIface(dev *water.Interface, ipClient net.IP) error {
	err := shared.ExecCmd("ip", "link", "set", "dev", dev.Name(), "mtu", fmt.Sprintf("%d", s.mtu), "up")
	if err != nil {
		return err
	}

	if !s.DoLocalIpConfig {
		return nil
	}

	return shared.ExecCmd("ip", "addr", "add", "dev", dev.Name(), fmt.Sprintf("%s/%d", s.VPNNet.GetServerIP().String(), s.VPNNet.GetSize()))
}

func (s *Server) getPlatformSpecifics(config *water.Config, ifaceConfig *InterfaceConfig) error {
	config.Name = ifaceConfig.Name
	config.Persist = ifaceConfig.Persist

	return nil
}

func (s *Server) verifyPlatformFlags() error {
	return nil
}
