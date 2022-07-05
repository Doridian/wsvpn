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

	if dev.IsTAP() {
		return shared.ExecCmd("ip", "addr", "add", "dev", dev.Name(), fmt.Sprintf("%s/%d", s.VPNNet.GetServerIP().String(), s.VPNNet.GetSize()))
	}

	return shared.ExecCmd("ip", "addr", "add", "dev", dev.Name(), s.VPNNet.GetServerIP().String(), "peer", ipClient.String())
}

func (s *Server) getPlatformSpecifics(config *water.Config, ifaceConfig *InterfacesConfig) error {
	if config.DeviceType == water.TAP {
		config.Name = ifaceConfig.Tap.Name
		config.Persist = ifaceConfig.Tap.Persist
	} else if ifaceConfig.Tun.NamePrefix != "" {
		config.Name = shared.FindLowestNetworkInterfaceByPrefix(ifaceConfig.Tun.NamePrefix)
	}
	return nil
}

func (s *Server) verifyPlatformFlags() error {
	return nil
}
