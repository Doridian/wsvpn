// +build linux
package main

import (
	"fmt"
	"github.com/Doridian/wsvpn/shared"
	"github.com/songgao/water"
	"net"
)

func configIface(dev *water.Interface, ipConfig bool, rNet *remoteNet, mtu int, routeGateway bool) error {
	err := shared.ExecCmd("ifconfig", dev.Name(), rNet.getClientIP(), "pointopoint", rNet.getServerIP(), "mtu", fmt.Sprintf("%d", mtu), "up")
	if err != nil {
		return err
	}
	err = shared.ExecCmd("ip", "route", "add", rNet.ipNet.String(), "via", rNet.getServerIP())
	if err != nil {
		return err
	}
	if routeGateway {
		err = shared.ExecCmd("ip", "route", "add", "default", "via", rNet.getServerIP())
		if err != nil {
			return err
		}
	}
	return nil
}

func getPlatformSpecifics(rNet *remoteNet, mtu int, config water.Config) water.Config {
	return config
}

func addRoute(dev *water.Interface, rNet *remoteNet, routeNet *net.IPNet) error {
	return shared.ExecCmd("ip", "route", "add", routeNet.String(), "via", rNet.getServerIP())
}
