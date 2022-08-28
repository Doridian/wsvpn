package iface

import (
	"fmt"
	"net"

	"github.com/Doridian/water"
	"github.com/Doridian/wsvpn/shared"
)

func inetFamily(ip net.IP) string {
	isIPv4 := ip.To4()
	if isIPv4 == nil {
		return "inet6"
	}
	return "inet"
}

func (w *WaterInterfaceWrapper) Configure(ipLocal net.IP, ipNet *shared.VPNNet, ipPeer net.IP) error {
	if ipLocal == nil {
		return shared.ExecCmd("ifconfig", w.Interface.Name(), "up")
	}

	ipNetSize, ipLocalCidr := w.splitSubnet(ipNet, ipLocal)

	inetType := inetFamily(ipLocal)

	var err error
	if w.Interface.IsTUN() && inetType == "inet" {
		err = shared.ExecCmd("ifconfig", w.Interface.Name(), inetType, ipLocalCidr, ipPeer.String(), "up")
	} else {
		err = shared.ExecCmd("ifconfig", w.Interface.Name(), inetType, ipLocalCidr, "up")
	}
	if err != nil {
		return err
	}

	return w.addSubnetRoute(ipNetSize, ipNet, ipPeer)
}

func (w *WaterInterfaceWrapper) AddInterfaceRoute(ipNet *net.IPNet) error {
	inetType := inetFamily(ipNet.IP)
	return shared.ExecCmd("route", "add", fmt.Sprintf("-%s", inetType), "-net", ipNet.String(), "-interface", w.Interface.Name())
}

func (w *WaterInterfaceWrapper) AddIPRoute(ipNet *net.IPNet, gateway net.IP) error {
	inetType := inetFamily(ipNet.IP)
	return shared.ExecCmd("route", "add", fmt.Sprintf("-%s", inetType), "-net", ipNet.String(), gateway.String())
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
