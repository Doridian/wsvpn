//go:build linux

package main

import (
	"flag"
	"fmt"
	"net"

	"github.com/Doridian/wsvpn/shared"
	"github.com/songgao/water"
)

var useTapName = flag.String("tap-name", "", "Use specific TAP name")
var useTapNoIfconfig = flag.Bool("tap-no-ifconfig", false, "Do not ifconfig the TAP")
var useTapPersist = flag.Bool("tap-persist", false, "Set persist on TAP")
var useTunNamePrefix = flag.String("tun-naming-prefix", "", "Use specific naming prefix for TUN interfaces, suffxed with a number starting at 0 (ex. wstun)")

func configIface(dev *water.Interface, ipConfig bool, mtu int, ipClient net.IP, ipServer net.IP, subnet *net.IPNet) error {
	if *useTapNoIfconfig {
		return nil
	}

	err := shared.ExecCmd("ip", "link", "set", "dev", dev.Name(), "mtu", fmt.Sprintf("%d", mtu), "up")
	if err != nil {
		return err
	}

	if !ipConfig {
		return nil
	}

	if dev.IsTAP() {
		subnetOnes, _ := subnet.Mask.Size()
		return shared.ExecCmd("ip", "addr", "add", "dev", dev.Name(), fmt.Sprintf("%s/%d", ipServer.String(), subnetOnes))
	}

	return shared.ExecCmd("ip", "addr", "add", "dev", dev.Name(), ipServer.String(), "peer", ipClient.String())
}

func extendTAPConfig(tapConfig *water.Config) error {
	tapName := *useTapName
	if tapName != "" {
		tapConfig.Name = tapName
	}
	tapConfig.Persist = *useTapPersist
	return nil
}

func extendTUNConfig(tunConfig *water.Config) error {
	tunNamePrefix := *useTunNamePrefix
	if tunNamePrefix != "" {
		tunConfig.Name = shared.FindLowestNetworkInterfaceByPrefix(tunNamePrefix)
	}

	return nil
}
