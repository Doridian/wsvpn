// +build linux
package main

import (
	"fmt"
	"github.com/Doridian/wsvpn/shared"
	"github.com/songgao/water"
	"net"
)

func configIface(dev *water.Interface, mtu int, ipClient net.IP, ipServer net.IP) error {
	err := shared.ExecCmd("ifconfig", dev.Name(), ipServer.String(), "pointopoint", ipClient.String(), "mtu", fmt.Sprintf("%d", mtu), "up")
	if err != nil {
		return err
	}
	return nil
}
