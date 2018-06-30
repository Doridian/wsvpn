// +build windows
package main

import (
	"fmt"
	"github.com/songgao/water"
	"os"
	"os/exec"
)

func configIface(dev *water.Interface, rNet *remoteNet, mtu string, routeGateway bool) error {
	gw := "gateway=none"
	if routeGateway {
		gw = fmt.Sprintf("gateway=%s", rNet.getServerIP())
	}
	cmd := exec.Command("netsh", "interface", "ip", "set", "address", "source=static", fmt.Sprintf("addr=%s", rNet.getClientIP()), fmt.Sprintf("name=%s", dev.Name()), fmt.Sprintf("mask=%s", rNet.getNetmask()), gw)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		print()
		return err
	}
	return nil
}

func getPlatformSpecifics(rNet *remoteNet, mtu string, config water.Config) water.Config {
	config.ComponentID = "tap0901"
	config.Network = rNet.str
	return config
}
