// +build darwin
package main

import (
	"fmt"
	"github.com/Doridian/wsvpn/shared"
	"github.com/songgao/water"
	"net"
)

func configIface(dev *water.Interface, ipConfig bool, rNet *remoteNet, mtu int, routeGateway bool) error {
	err := shared.ExecCmd("ifconfig", dev.Name(), "mtu", fmt.Sprintf("%d", mtu))
	if err != nil {
		return err
	}
	if !ipConfig {
		return shared.ExecCmd("ifconfig", dev.Name(), "up")
	}

	err = shared.ExecCmd("ifconfig", dev.Name(), rNet.getClientIP(), rNet.getServerIP(), "up")
	if err != nil {
		return err
	}
	err = shared.ExecCmd("route", "add", "-net", rNet.ipNet.String(), "gw", rNet.getServerIP())
	if err != nil {
		return err
	}
	if routeGateway {
		err = shared.ExecCmd("route", "add", "default", "gw", rNet.getServerIP())
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
	return shared.ExecCmd("route", "add", "-net", routeNet.String(), "gw", rNet.getServerIP())
}
