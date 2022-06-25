//go:build windows

package clients

import (
	"flag"
	"fmt"
	"net"

	"github.com/Doridian/wsvpn/shared"
	"github.com/songgao/water"
)

var ifaceComponentID = flag.String("interface-component-id", "tap0901", "Commponent ID of the interface to use") // TODO: No flag parsing in Client

func (c *Client) configureInterfaceMTU() error {
	return shared.ExecCmd("netsh", "interface", "ipv4", "set", "subinterface", c.iface.Name(), fmt.Sprintf("mtu=%d", c.mtu))
}

func (c *Client) configureInterface() error {
	err := c.configureInterfaceMTU()
	if err != nil {
		return err
	}

	if !c.doIpConfig {
		return shared.ExecCmd("netsh", "interface", "ip", "set", "address", "source=dhcp", fmt.Sprintf("name=%s", c.iface.Name()))
	}

	gw := "gateway=none"
	if c.SetDefaultGateway {
		gw = fmt.Sprintf("gateway=%s", c.remoteNet.GetServerIP())
	}
	return shared.ExecCmd("netsh", "interface", "ip", "set", "address", "source=static", fmt.Sprintf("addr=%s", c.remoteNet.GetRaw()), fmt.Sprintf("name=%s", c.iface.Name()), fmt.Sprintf("mask=%s", c.remoteNet.GetNetmask()), gw)
}

func (c *Client) getPlatformSpecifics(config water.Config) water.Config {
	config.ComponentID = *ifaceComponentID
	config.Network = c.remoteNet.GetRaw()
	config.InterfaceName = c.InterfaceName
	return config
}

func (c *Client) addRoute(routeNet *net.IPNet) error {
	return shared.ExecCmd("route", "ADD", routeNet.String(), c.remoteNet.GetServerIP().String())
}
