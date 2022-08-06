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

func (s *Server) getPlatformSpecifics(config *water.Config) error {
	config.ComponentID = s.InterfaceConfig.ComponentId
	config.InterfaceName = s.InterfaceConfig.Name

	return nil
}

func (s *Server) verifyPlatformFlags() error {
	if !s.InterfaceConfig.OneInterfacePerConnection {
		return errors.New("windows can not enable one-interface-per-connection")
	}
	return nil
}
