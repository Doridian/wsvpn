// +build darwin
package main

import (
	"fmt"
	"github.com/songgao/water"
	"net"
)

func configIface(dev *water.Interface, mtu int, ipClient net.IP, ipServer net.IP) error {
	err := execCmd("ifconfig", dev.Name(), "mtu", fmt.Sprintf("%d", mtu))
	if err != nil {
		return err
	}
	err = execCmd("ifconfig", dev.Name(), ipServer.String(), ipClient.String(), "up")
	if err != nil {
		return err
	}
	return nil
}
