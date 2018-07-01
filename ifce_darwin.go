// +build darwin
package main

import (
	"github.com/songgao/water"
	"net"
	"os/exec"
)

func configIface(dev *water.Interface, mtu string, ipClient net.IP, ipServer net.IP) error {
	err := exec.Command("ifconfig", dev.Name(), "mtu", mtu).Run()
	if err != nil {
		return err
	}
	err = exec.Command("ifconfig", dev.Name(), ipServer.String(), ipClient.String(), "up").Run()
	if err != nil {
		return err
	}
	return nil
}
