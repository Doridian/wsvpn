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

	ipLocalV4 := ipLocal.To4()
	if ipLocalV4 != nil {
		ipLocal = ipLocalV4
	}
	ipNetSize, _ := w.splitSubnet(ipNet, ipLocal)

	var err error
	if ipLocalV4 != nil {
		ipMask := net.CIDRMask(ipNetSize, len(ipLocal)*8)
		ipMaskStr := shared.IPMaskGetNetMask(ipMask)

		err = shared.ExecCmd("netsh", "interface", "ipv4", "set", "address", "store=active", "source=static", "gateway=none", fmt.Sprintf("addr=%s", ipLocal), fmt.Sprintf("name=%s", w.Interface.Name()), fmt.Sprintf("mask=%s", ipMaskStr))
	} else {
		err = shared.ExecCmd("netsh", "interface", "ipv6", "set", "address", "store=active", fmt.Sprintf("addr=%s", ipLocal), fmt.Sprintf("interface=%s", w.Interface.Name()))
	}
	if err != nil {
		return err
	}

	return w.addSubnetRoute(ipNetSize, ipNet, ipPeer)
}

func (w *WaterInterfaceWrapper) AddInterfaceRoute(ipNet *net.IPNet) error {
	ipNetV4 := ipNet.IP.To4()
	if ipNetV4 != nil {
		return w.AddIPRoute(ipNet, net.IPv4zero)
	} else {
		return w.AddIPRoute(ipNet, net.IPv6zero)
	}
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

	config.ComponentID = ifaceConfig.ComponentID

	return nil
}

func VerifyPlatformFlags(ifaceConfig *InterfaceConfig, mode shared.VPNMode) error {
	if ifaceConfig.OneInterfacePerConnection {
		return errors.New("windows can not support one-interface-per-connection")
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
	defer func() {
		_ = fh.Close()
	}()

	_, err = fh.Write(wintunDll)
	return err
}
