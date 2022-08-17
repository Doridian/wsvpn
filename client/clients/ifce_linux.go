//go:build linux

package clients

import (
	"fmt"
	"net"

	"github.com/Doridian/water"
	"github.com/Doridian/wsvpn/shared"
)

func (c *Client) configureInterfaceMTU() error {
	return shared.ExecCmd("ip", "link", "set", "dev", c.iface.Name(), "mtu", fmt.Sprintf("%d", c.mtu))
}

func (c *Client) configureInterface() error {
	err := shared.ExecCmd("ip", "link", "set", "dev", c.iface.Name(), "mtu", fmt.Sprintf("%d", c.mtu), "up")
	if err != nil {
		return err
	}

	if !c.doIpConfig {
		return nil
	}

	err = shared.ExecCmd("ip", "addr", "add", "dev", c.iface.Name(), c.remoteNet.GetRawIP().String(), "peer", c.remoteNet.GetServerIP().String())
	if err != nil {
		return err
	}

	if c.SetDefaultGateway {
		err = shared.ExecCmd("ip", "route", "add", "default", "via", c.remoteNet.GetServerIP().String())
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Client) getPlatformSpecifics(config *water.Config, ifaceConfig *shared.InterfaceConfig) error {
	config.Name = ifaceConfig.Name
	config.Persist = ifaceConfig.Persist
	return nil
}

func (c *Client) addRoute(routeNet *net.IPNet) error {
	return shared.ExecCmd("ip", "route", "add", routeNet.String(), "via", c.remoteNet.GetServerIP().String())
}
