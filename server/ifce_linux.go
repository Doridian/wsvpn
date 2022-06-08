//go:build linux
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
var useTapNoIfconfig = flag.Bool("tap-no-ifconfig", false, "Do not ifconfig the TAP")
var useTapPersist = flag.Bool("tap-persist", false, "Set persist on TAP")

func configIface(dev *water.Interface, ipConfig bool, mtu int, ipClient net.IP, ipServer net.IP, subnet *net.IPNet) error {
	if *useTapNoIfconfig {
		return nil
	}

	if !ipConfig {
		return shared.ExecCmd("ifconfig", dev.Name(), "mtu", fmt.Sprintf("%d", mtu), "up")
	}

	if dev.IsTAP() {
		subnetOnes, _ := subnet.Mask.Size()
		return shared.ExecCmd("ifconfig", dev.Name(), fmt.Sprintf("%s/%d", ipServer.String(), subnetOnes), "mtu", fmt.Sprintf("%d", mtu), "up")
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
