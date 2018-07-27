// +build darwin
package main

import (
	"fmt"
	"github.com/songgao/water"
	"net"
)

func configIface(dev *water.Interface, rNet *remoteNet, mtu int, routeGateway bool) error {
	err := execCmd("ifconfig", dev.Name(), "mtu", fmt.Sprintf("%d", mtu))
	if err != nil {
		return err
	}
	err = execCmd("ifconfig", dev.Name(), rNet.getClientIP(), rNet.getServerIP(), "up")
	if err != nil {
		return err
	}
	err = execCmd("route", "add", "-net", rNet.ipNet.String(), "gw", rNet.getServerIP())
	if err != nil {
		return err
	}
	if routeGateway {
		err = execCmd("route", "add", "default", "gw", rNet.getServerIP())
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
	return execCmd("route", "add", "-net", routeNet.String(), "gw", rNet.getServerIP())
}
