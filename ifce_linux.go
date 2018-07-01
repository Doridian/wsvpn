// +build linux
package main

import (
	"fmt"
	"github.com/songgao/water"
)

func configIface(dev *water.Interface, rNet *remoteNet, mtu int, routeGateway bool) error {
	err := execCmd("ifconfig", dev.Name(), rNet.getClientIP(), "pointopoint", rNet.getServerIP(), "mtu", fmt.Sprintf("%d", mtu), "up")
	if err != nil {
		return err
	}
	err = execCmd("ip", "route", "add", rNet.ipNet.String(), "via", rNet.getServerIP())
	if err != nil {
		return err
	}
	return nil
}

func getPlatformSpecifics(rNet *remoteNet, mtu int, config water.Config) water.Config {
	return config
}
