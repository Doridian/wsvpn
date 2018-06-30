// +build linux
package main

import (
	"github.com/songgao/water"
	"os/exec"
)

func configIface(dev *water.Interface, rNet *remoteNet, mtu string, routeGateway bool) error {
	err := exec.Command("ifconfig", dev.Name(), rNet.getServerIP(), "pointopoint", rNet.getClientIP(), "mtu", mtu, "up").Run()
	if err != nil {
		return err
	}
	return nil
}

func getPlatformSpecifics(rNet *remoteNet, mtu string, config water.Config) water.Config {
	return config
}
