// +build windows
package main

import (
	"flag"
	"fmt"
	"github.com/Doridian/wsvpn/shared"
	"github.com/songgao/water"
	"net"
)

var ifaceName = flag.String("ifname", "tap0901", "Name of the interface to use")

func configIface(dev *water.Interface, rNet *remoteNet, mtu int, routeGateway bool) error {
	gw := "gateway=none"
	if routeGateway {
		gw = fmt.Sprintf("gateway=%s", rNet.getServerIP())
	}
	err := shared.ExecCmd("netsh", "interface", "ip", "set", "address", "source=static", fmt.Sprintf("addr=%s", rNet.getClientIP()), fmt.Sprintf("name=%s", dev.Name()), fmt.Sprintf("mask=%s", rNet.getNetmask()), gw)
	if err != nil {
		return err
	}
	return nil
}

func getPlatformSpecifics(rNet *remoteNet, mtu int, config water.Config) water.Config {
	config.ComponentID = *ifaceName
	config.Network = rNet.str
	return config
}

func addRoute(dev *water.Interface, rNet *remoteNet, routeNet *net.IPNet) error {
	return shared.ExecCmd("route", "ADD", routeNet.String(), rNet.getServerIP())
}
