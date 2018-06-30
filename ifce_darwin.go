// +build darwin
package main

import (
	"github.com/songgao/water"
	"os/exec"
)

func configIface(dev *water.Interface, rNet *remoteNet, mtu string, routeGateway bool) error {
	err := exec.Command("ifconfig", dev.Name(), "mtu", mtu).Run()
	if err != nil {
		return err
	}
	err = exec.Command("ifconfig", dev.Name(), rNet.getServerIP(), rNet.getClientIP(), "up").Run()
	if err != nil {
		return err
	}
	return nil
}

func getPlatformSpecifics(rNet *remoteNet, mtu string, config water.Config) water.Config {
	return config
}
