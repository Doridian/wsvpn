// +build linux
package main

import (
	"fmt"
	"github.com/songgao/water"
	"net"
)

func configIface(dev *water.Interface, mtu int, ipClient net.IP, ipServer net.IP) error {
	err := execCmd("ifconfig", dev.Name(), ipServer.String(), "pointopoint", ipClient.String(), "mtu", fmt.Sprintf("%d", mtu), "up")
	if err != nil {
		return err
	}
	return nil
}
