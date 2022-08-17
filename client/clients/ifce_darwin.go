//go:build darwin

package clients

import (
	"fmt"
	"net"

	"github.com/Doridian/water"
	"github.com/Doridian/wsvpn/shared"
)

func (c *Client) configureInterfaceMTU() error {
	return shared.ExecCmd("ifconfig", c.iface.Name(), "mtu", fmt.Sprintf("%d", c.mtu))
}

func (c *Client) configureInterface() error {
	err := c.configureInterfaceMTU()
	if err != nil {
		return err
	}

	if !c.doIpConfig {
		return shared.ExecCmd("ifconfig", c.iface.Name(), "up")
	}

	err = shared.ExecCmd("ifconfig", c.iface.Name(), c.remoteNet.GetRawIP().String(), c.remoteNet.GetServerIP().String(), "up")
	if err != nil {
		return err
	}

	if c.SetDefaultGateway {
		err = shared.ExecCmd("route", "add", "default", c.remoteNet.GetServerIP().String())
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Client) getPlatformSpecifics(config *water.Config, ifaceConfig *shared.InterfaceConfig) error {
	config.Name = ifaceConfig.Name
	return nil
}

func (c *Client) addRoute(routeNet *net.IPNet) error {
	return shared.ExecCmd("route", "add", "-net", routeNet.String(), c.remoteNet.GetServerIP().String())
}
