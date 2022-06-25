//go:build windows

package servers

import (
	"errors"
	"flag"
	"fmt"
	"net"

	"github.com/Doridian/wsvpn/shared"
	"github.com/songgao/water"
)

var useTapName = flag.String("tap-name", "", "Use specific TAP name")                               // TODO: No flag parsing in Server
var useTapComponentID = flag.String("tap-component-id", "tap0901", "Use specific TAP component ID") // TODO: No flag parsing in Server

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

func (s *Server) extendTAPConfig(config *water.Config) error {
	config.ComponentID = *useTapComponentID
	config.InterfaceName = *useTapName
	return nil
}

func (s *Server) extendTUNConfig(tunConfig *water.Config) error {
	return errors.New("running the server on Windows requires using TAP mode")
}

func (s *Server) verifyPlatformFlags() error {
	if s.Mode != shared.VPN_MODE_TAP {
		return errors.New("running the server on Windows requires using TAP mode")
	}
	return nil
}
