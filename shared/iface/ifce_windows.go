package iface

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"

	"github.com/Doridian/water"
	"github.com/Doridian/wsvpn/shared"
)

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

	return w.addSubnetRoute(ipNetSize, ipNet, ipPeer)
}

func (w *WaterInterfaceWrapper) AddInterfaceRoute(ipNet *net.IPNet) error {
	return w.AddIPRoute(ipNet, net.IPv4(0, 0, 0, 0))
}

func (w *WaterInterfaceWrapper) AddIPRoute(ipNet *net.IPNet, gateway net.IP) error {
	iface, err := w.GetNetInterface()
	if err != nil {
		return err
	}
	return shared.ExecCmd("route", "ADD", ipNet.String(), gateway.String(), "IF", fmt.Sprintf("%d", iface.Index))
}

func GetPlatformSpecifics(config *water.Config, ifaceConfig *InterfaceConfig) error {
	setName := getInterfaceNameOrPrefix(ifaceConfig)
	if setName != "" {
		config.InterfaceName = setName
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

func InitializeWater() error {
	if len(wintunDll) < 1 {
		return errors.New("could not find embedded wintun.dll")
	}

	execFileName, err := os.Executable()
	if err != nil {
		return err
	}

	execFilePath := filepath.Dir(execFileName)

	fh, err := os.Create(filepath.Join(execFilePath, "wintun.dll"))
	if err != nil {
		return err
	}
	fh.Write(wintunDll)
	fh.Close()
	return nil
}
