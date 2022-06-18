//go:build darwin

package main

import (
	"fmt"
	"net"

	"github.com/Doridian/wsvpn/shared"
	"github.com/songgao/water"
)

func configIface(dev *water.Interface, ipConfig bool, mtu int, ipClient net.IP, ipServer net.IP, subnet *net.IPNet) error {
	err := shared.ExecCmd("ifconfig", dev.Name(), "mtu", fmt.Sprintf("%d", mtu))
	if err != nil {
		return err
	}

	if !ipConfig {
		return shared.ExecCmd("ifconfig", dev.Name(), "up")
	}

	if dev.IsTAP() {
		subnetOnes, _ := subnet.Mask.Size()
		return shared.ExecCmd("ifconfig", dev.Name(), fmt.Sprintf("%s/%d", ipServer.String(), subnetOnes), "up")
	}
	return shared.ExecCmd("ifconfig", dev.Name(), ipServer.String(), ipClient.String(), "up")
}

func extendTAPConfig(config *water.Config) error {
	return nil
}

func extendTUNConfig(tunConfig *water.Config) error {
	return nil
}
