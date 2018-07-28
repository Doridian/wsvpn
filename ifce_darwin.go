// +build darwin
package main

import (
	"fmt"
	"github.com/Doridian/wstun_shared"
	"github.com/songgao/water"
	"net"
)

func configIface(dev *water.Interface, mtu int, ipClient net.IP, ipServer net.IP) error {
	err := wstun_shared.ExecCmd("ifconfig", dev.Name(), "mtu", fmt.Sprintf("%d", mtu))
	if err != nil {
		return err
	}
	err = wstun_shared.ExecCmd("ifconfig", dev.Name(), ipServer.String(), ipClient.String(), "up")
	if err != nil {
		return err
	}
	return nil
}
