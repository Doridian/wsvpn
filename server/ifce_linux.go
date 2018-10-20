// +build linux
package main

import (
	"flag"
	"fmt"
	"github.com/Doridian/wsvpn/shared"
	"github.com/songgao/water"
	"net"
)

var useTapName = flag.String("tap-name", "", "Use specific TAP name")
var useTapPersist = flag.Bool("tap-persist", false, "Set persist on TAP")

func configIface(dev *water.Interface, ipConfig bool, mtu int, ipClient net.IP, ipServer net.IP) error {
	if *useTapPersist {
		return nil
	}

	if !ipConfig {
		return shared.ExecCmd("ifconfig", dev.Name(), "mtu", fmt.Sprintf("%d", mtu), "up")
	}

	if dev.IsTAP() {
		return shared.ExecCmd("ifconfig", dev.Name(), ipServer.String(), "mtu", fmt.Sprintf("%d", mtu), "up")
	}
	return shared.ExecCmd("ifconfig", dev.Name(), ipServer.String(), "pointopoint", ipClient.String(), "mtu", fmt.Sprintf("%d", mtu), "up")
}

func extendTAPConfig(tapConfig *water.Config) {
	tapName := *useTapName
	if tapName != "" {
		tapConfig.Name = tapName
	}
	tapConfig.Persist = *useTapPersist
}
