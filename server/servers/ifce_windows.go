//go:build windows

package servers

import (
	"errors"
	"fmt"
	"net"

	"github.com/Doridian/wsvpn/shared"
	"github.com/songgao/water"
)

func (s *Server) configIface(dev *water.Interface, ipClient net.IP) error {
	err := shared.ExecCmd("netsh", "interface", "ipv4", "set", "subinterface", dev.Name(), fmt.Sprintf("mtu=%d", s.mtu))
	if err != nil {
		return err
	}

	if !s.DoLocalIpConfig {
		return shared.ExecCmd("netsh", "interface", "ip", "set", "address", "source=dhcp", fmt.Sprintf("name=%s", dev.Name()))
	}

	return shared.ExecCmd("netsh", "interface", "ip", "set", "address", "source=static", "gateway=none", fmt.Sprintf("addr=%s", s.VPNNet.GetServerIP().String()), fmt.Sprintf("name=%s", dev.Name()), fmt.Sprintf("mask=%s", shared.IPNetGetNetMask(s.VPNNet.GetSubnet())))
}

func (s *Server) getPlatformSpecifics(config *water.Config, ifaceConfig *InterfacesConfig) error {
	if config.DeviceType == water.TAP {
		config.ComponentID = ifaceConfig.Tap.ComponentId
		config.InterfaceName = ifaceConfig.Tap.Name
	}

	return nil
}

func (s *Server) verifyPlatformFlags() error {
	if s.Mode != shared.VPN_MODE_TAP {
		return errors.New("running the server on Windows requires using TAP mode")
	}
	return nil
}
