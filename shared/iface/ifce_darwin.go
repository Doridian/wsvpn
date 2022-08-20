package iface

import (
	"net"

	"github.com/Doridian/water"
	"github.com/Doridian/wsvpn/shared"
)

func (w *WaterInterfaceWrapper) Configure(ipLocal net.IP, ipNet *shared.VPNNet, ipPeer net.IP) error {
	if ipLocal == nil {
		return shared.ExecCmd("ifconfig", w.Interface.Name(), "up")
	}

	ipNetSize, ipLocalCidr := w.splitSubnet(ipNet, ipLocal)

	var err error
	if w.Interface.IsTUN() {
		err = shared.ExecCmd("ifconfig", w.Interface.Name(), ipLocalCidr, ipPeer.String(), "up")
	} else {
		err = shared.ExecCmd("ifconfig", w.Interface.Name(), ipLocalCidr, "up")
	}
	if err != nil {
		return err
	}

	return w.addSubnetRoute(ipNetSize, ipNet, ipPeer)
}

func (w *WaterInterfaceWrapper) AddInterfaceRoute(ipNet *net.IPNet) error {
	return shared.ExecCmd("route", "add", "-net", ipNet.String(), "-interface", w.Interface.Name())
}

func (w *WaterInterfaceWrapper) AddIPRoute(ipNet *net.IPNet, gateway net.IP) error {
	return shared.ExecCmd("route", "add", "-net", ipNet.String(), gateway.String())
}

func GetPlatformSpecifics(config *water.Config, ifaceConfig *InterfaceConfig) error {
	setName := getInterfaceNameOrPrefix(ifaceConfig)
	if setName != "" {
		config.Name = setName
	}

	return nil
}

func VerifyPlatformFlags(ifaceConfig *InterfaceConfig, mode shared.VPNMode) error {
	return nil
}

func InitializeWater() error {
	return nil
}
