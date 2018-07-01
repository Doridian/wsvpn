// +build darwin
package main

import (
	"fmt"
	"github.com/songgao/water"
	"os/exec"
)

func configIface(dev *water.Interface, rNet *remoteNet, mtu int, routeGateway bool) error {
	err := exec.Command("ifconfig", dev.Name(), "mtu", fmt.Sprintf("%d", mtu)).Run()
	if err != nil {
		return err
	}
	err = exec.Command("ifconfig", dev.Name(), rNet.getClientIP(), rNet.getServerIP(), "up").Run()
	if err != nil {
		return err
	}
	err = exec.Command("route", "add", "-net", rNet.ipNet.String(), "gw", rNet.getServerIP())
	if err != nil {
		return err
	}
	return nil
}

func getPlatformSpecifics(rNet *remoteNet, mtu int, config water.Config) water.Config {
	return config
}
