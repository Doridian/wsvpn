// +build linux
package main

import (
	"fmt"
	"github.com/songgao/water"
	"net"
	"os/exec"
)

func configIface(dev *water.Interface, mtu int, ipClient net.IP, ipServer net.IP) error {
	err := exec.Command("ifconfig", dev.Name(), ipServer.String(), "pointopoint", ipClient.String(), "mtu", fmt.Sprintf("%d", mtu), "up").Run()
	if err != nil {
		return err
	}
	return nil
}
