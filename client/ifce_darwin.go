// +build darwin
package main

import (
	"fmt"
	"github.com/songgao/water"
	"net"
)

func configIface(dev *water.Interface, rNet *remoteNet, mtu int, routeGateway bool) error {
	err := wstun_shared.ExecCmd("ifconfig", dev.Name(), "mtu", fmt.Sprintf("%d", mtu))
	if err != nil {
		return err
	}
	err = wstun_shared.ExecCmd("ifconfig", dev.Name(), rNet.getClientIP(), rNet.getServerIP(), "up")
	if err != nil {
		return err
	}
	err = wstun_shared.ExecCmd("route", "add", "-net", rNet.ipNet.String(), "gw", rNet.getServerIP())
	if err != nil {
		return err
	}
	if routeGateway {
		err = wstun_shared.ExecCmd("route", "add", "default", "gw", rNet.getServerIP())
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
	return wstun_shared.ExecCmd("route", "add", "-net", routeNet.String(), "gw", rNet.getServerIP())
}
