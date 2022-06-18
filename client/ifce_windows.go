//go:build windows

package main

import (
	"flag"
	"fmt"
	"net"

	"github.com/Doridian/wsvpn/shared"
	"github.com/songgao/water"
)

var ifaceComponentID = flag.String("interface-component-id", "tap0901", "Commponent ID of the interface to use")

func configIface(dev *water.Interface, ipConfig bool, rNet *remoteNet, mtu int, routeGateway bool) error {
	err := shared.ExecCmd("netsh", "interface", "ipv4", "set", "subinterface", dev.Name(), fmt.Sprintf("mtu=%d", mtu))
	if err != nil {
		return err
	}

	if !ipConfig {
		return shared.ExecCmd("netsh", "interface", "ip", "set", "address", "source=dhcp", fmt.Sprintf("name=%s", dev.Name()))
	}

	gw := "gateway=none"
	if routeGateway {
		gw = fmt.Sprintf("gateway=%s", rNet.getServerIP())
	}
	return shared.ExecCmd("netsh", "interface", "ip", "set", "address", "source=static", fmt.Sprintf("addr=%s", rNet.getClientIP()), fmt.Sprintf("name=%s", dev.Name()), fmt.Sprintf("mask=%s", rNet.getNetmask()), gw)
}

func getPlatformSpecifics(rNet *remoteNet, mtu int, name string, config water.Config) water.Config {
	config.ComponentID = *ifaceComponentID
	config.Network = rNet.str
	config.InterfaceName = name
	return config
}

func addRoute(dev *water.Interface, rNet *remoteNet, routeNet *net.IPNet) error {
	return shared.ExecCmd("route", "ADD", routeNet.String(), rNet.getServerIP())
}
