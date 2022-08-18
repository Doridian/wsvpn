//go:build linux

package iface

import (
	"fmt"
	"net"

	"github.com/Doridian/water"
	"github.com/Doridian/wsvpn/shared"
)

func (w *WaterInterfaceWrapper) SetMTU(mtu int) error {
	return shared.ExecCmd("ip", "link", "set", "dev", w.Interface.Name(), "mtu", fmt.Sprintf("%d", mtu))
}

func (w *WaterInterfaceWrapper) Configure(ipLocal net.IP, ipNet *shared.VPNNet, ipPeer net.IP) error {
	err := shared.ExecCmd("ip", "link", "set", "dev", w.Interface.Name(), "up")
	if err != nil {
		return err
	}

	if ipLocal == nil {
		return nil
	}

	ipNetSize, ipLocalCidr := w.splitSubnet(ipNet, ipLocal)

	err = shared.ExecCmd("ip", "addr", "add", "dev", w.Interface.Name(), ipLocalCidr, "peer", ipPeer.String())
	if err != nil {
		err = shared.ExecCmd("ip", "addr", "add", "dev", w.Interface.Name(), ipLocalCidr)
	}

	if err != nil {
		return err
	}

	return w.addPeerRoute(ipNetSize, ipPeer)
}

func (w *WaterInterfaceWrapper) AddInterfaceRoute(ipNet *net.IPNet) error {
	return shared.ExecCmd("ip", "route", "add", ipNet.String(), "dev", w.Interface.Name())
}

func (w *WaterInterfaceWrapper) AddIPRoute(ipNet *net.IPNet, gateway net.IP) error {
	return shared.ExecCmd("ip", "route", "add", ipNet.String(), "via", gateway.String())
}

func GetPlatformSpecifics(config *water.Config, ifaceConfig *InterfaceConfig) error {
	setName := getInterfaceNameOrPrefix(ifaceConfig)
	if setName != "" {
		config.Name = ifaceConfig.Name
	}

	config.Persist = ifaceConfig.Persist

	return nil
}

func VerifyPlatformFlags(ifaceConfig *InterfaceConfig, mode shared.VPNMode) error {
	return nil
}

func InitializeWater() error {
	return nil
}
