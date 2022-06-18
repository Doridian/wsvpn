//go:build linux

package main

import (
	"fmt"
	"net"

	"github.com/Doridian/wsvpn/shared"
	"github.com/songgao/water"
)

func configIface(dev *water.Interface, ipConfig bool, rNet *remoteNet, mtu int, routeGateway bool) error {
	err := shared.ExecCmd("ip", "link", "set", "dev", dev.Name(), "mtu", fmt.Sprintf("%d", mtu), "up")
	if err != nil {
		return err
	}

	err = shared.ExecCmd("ip", "addr", "add", "dev", dev.Name(), rNet.getClientIP(), "peer", rNet.getServerIP())
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

func getPlatformSpecifics(rNet *remoteNet, mtu int, name string, config water.Config) water.Config {
	if name != "" {
		config.Name = name
	}
	return config
}

func addRoute(dev *water.Interface, rNet *remoteNet, routeNet *net.IPNet) error {
	return shared.ExecCmd("ip", "route", "add", routeNet.String(), "via", rNet.getServerIP())
}
