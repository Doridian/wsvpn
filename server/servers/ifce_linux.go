//go:build linux

package servers

import (
	"flag"
	"fmt"
	"net"

	"github.com/Doridian/wsvpn/shared"
	"github.com/songgao/water"
)

var useTapName = flag.String("tap-name", "", "Use specific TAP name")                                                                                                         // TODO: No flag parsing in Server
var useTapPersist = flag.Bool("tap-persist", false, "Set persist on TAP")                                                                                                     // TODO: No flag parsing in Server
var useTunNamePrefix = flag.String("tun-naming-prefix", "", "Use specific naming prefix for TUN interfaces (e.g. wstun), automatically suffixed with a number starting at 0") // TODO: No flag parsing in Server

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

func (s *Server) extendTAPConfig(tapConfig *water.Config) error {
	tapName := *useTapName
	if tapName != "" {
		tapConfig.Name = tapName
	}
	tapConfig.Persist = *useTapPersist
	return nil
}

func (s *Server) extendTUNConfig(tunConfig *water.Config) error {
	tunNamePrefix := *useTunNamePrefix
	if tunNamePrefix != "" {
		tunConfig.Name = shared.FindLowestNetworkInterfaceByPrefix(tunNamePrefix)
	}

	return nil
}

func (s *Server) verifyPlatformFlags() error {
	return nil
}
