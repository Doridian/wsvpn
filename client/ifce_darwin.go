//go:build darwin

package main

import (
	"fmt"
	"net"

	"github.com/Doridian/wsvpn/shared"
	"github.com/songgao/water"
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
	err = shared.ExecCmd("route", "add", "-net", rNet.ipNet.String(), rNet.getServerIP())
	if err != nil {
		return err
	}
	if routeGateway {
		err = shared.ExecCmd("route", "add", "default", rNet.getServerIP())
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
	return shared.ExecCmd("route", "add", "-net", routeNet.String(), rNet.getServerIP())
}
