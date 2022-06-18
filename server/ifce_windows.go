//go:build windows

package main

import (
	"errors"
	"flag"
	"fmt"
	"net"

	"github.com/Doridian/wsvpn/shared"
	"github.com/songgao/water"
)

var useTapName = flag.String("tap-name", "", "Use specific TAP name")
var useTapComponentID = flag.String("tap-component-id", "tap0901", "Use specific TAP component ID")

func configIface(dev *water.Interface, ipConfig bool, mtu int, ipClient net.IP, ipServer net.IP, subnet *net.IPNet) error {
	err := shared.ExecCmd("netsh", "interface", "ipv4", "set", "subinterface", dev.Name(), fmt.Sprintf("mtu=%d", mtu))
	if err != nil {
		return err
	}

	if !ipConfig {
		return shared.ExecCmd("netsh", "interface", "ip", "set", "address", "source=dhcp", fmt.Sprintf("name=%s", dev.Name()))
	}

	return shared.ExecCmd("netsh", "interface", "ip", "set", "address", "source=static", "gateway=none", fmt.Sprintf("addr=%s", ipServer.String()), fmt.Sprintf("name=%s", dev.Name()), fmt.Sprintf("mask=%s", shared.IPNetGetNetMask(subnet)))
}

func extendTAPConfig(config *water.Config) error {
	config.ComponentID = *useTapComponentID
	config.InterfaceName = *useTapName
	return nil
}

func extendTUNConfig(tunConfig *water.Config) error {
	return errors.New("running the server on Windows requires using TAP mode")
}
