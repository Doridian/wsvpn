//go:build windows

package iface

import (
	"errors"
	"fmt"
	"log"
	"net"

	"github.com/Doridian/water"
	"github.com/Doridian/wsvpn/shared"
)

func (w *WaterInterfaceWrapper) SetMTU(mtu int) error {
	log.Printf("ForceMTU(): %v", w.Interface.ForceMTU(mtu))
	return shared.ExecCmd("netsh", "interface", "ipv4", "set", "subinterface", w.Interface.Name(), fmt.Sprintf("mtu=%d", mtu))
}

func (w *WaterInterfaceWrapper) Configure(ipLocal net.IP, ipNet *shared.VPNNet, ipPeer net.IP) error {
	if ipLocal == nil {
		return shared.ExecCmd("netsh", "interface", "ip", "set", "address", "source=dhcp", fmt.Sprintf("name=%s", w.Interface.Name()))
	}

	ipNetSize, _ := w.splitSubnet(ipNet, ipLocal)
	ipMask := net.CIDRMask(ipNetSize, 32)
	ipMaskStr := fmt.Sprintf("%d.%d.%d.%d", ipMask[0], ipMask[1], ipMask[2], ipMask[3])

	err := shared.ExecCmd("netsh", "interface", "ip", "set", "address", "source=static", "gateway=none", fmt.Sprintf("addr=%s", ipLocal), fmt.Sprintf("name=%s", w.Interface.Name()), fmt.Sprintf("mask=%s", ipMaskStr))

	if err != nil {
		return err
	}

	return w.addPeerRoute(ipNetSize, ipPeer)
}

func (w *WaterInterfaceWrapper) AddInterfaceRoute(ipNet *net.IPNet) error {
	return shared.ExecCmd("route", "ADD", ipNet.String(), "IF", w.Interface.Name())
}

func (w *WaterInterfaceWrapper) AddIPRoute(ipNet *net.IPNet, gateway net.IP) error {
	return shared.ExecCmd("route", "ADD", ipNet.String(), gateway.String())
}

func GetPlatformSpecifics(config *water.Config, ifaceConfig *InterfaceConfig) error {
	setName := getInterfaceNameOrPrefix(ifaceConfig)
	if setName != "" {
		config.InterfaceName = ifaceConfig.Name
	}

	config.ComponentID = ifaceConfig.ComponentId

	return nil
}

func VerifyPlatformFlags(ifaceConfig *InterfaceConfig, mode shared.VPNMode) error {
	if !ifaceConfig.OneInterfacePerConnection && mode == shared.VPN_MODE_TAP {
		return errors.New("Windows can not use one-interface-per-connection with TAP")
	}

	return nil
}
