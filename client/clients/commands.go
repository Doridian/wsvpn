package clients

import (
	"encoding/json"
	"errors"
	"log"
	"net"

	"github.com/Doridian/wsvpn/shared"
	"github.com/Doridian/wsvpn/shared/commands"
	"github.com/google/uuid"
	"github.com/songgao/water"
)

func (c *Client) registerCommandHandlers() {
	c.socket.AddCommandHandler(commands.AddRouteCommandName, func(command *commands.IncomingCommand) error {
		var err error
		var parameters commands.AddRouteParameters
		err = json.Unmarshal(command.Parameters, &parameters)
		if err != nil {
			return err
		}

		if c.iface == nil || c.remoteNet == nil {
			return errors.New("cannot addroute before init")
		}

		_, routeNet, err := net.ParseCIDR(parameters.Route)
		if err != nil {
			return err
		}

		return c.addRoute(routeNet)
	})

	c.socket.AddCommandHandler(commands.InitCommandName, func(command *commands.IncomingCommand) error {
		var err error
		var parameters commands.InitParameters
		err = json.Unmarshal(command.Parameters, &parameters)
		if err != nil {
			return err
		}

		if parameters.ClientID == "" {
			clientUUID, err := uuid.NewRandom()
			if err != nil {
				return err
			}
			c.ClientID = clientUUID.String()
		} else {
			c.ClientID = parameters.ClientID
		}

		mode := shared.VPNModeFromString(parameters.Mode)

		c.remoteNet, err = shared.ParseVPNNet(parameters.IpAddress)
		if err != nil {
			return err
		}

		c.doIpConfig = parameters.DoIpConfig

		log.Printf("[%s] Network mode %s, subnet %s, mtu %d, IPConfig %s", c.socket.GetConnectionID(), parameters.Mode, c.remoteNet.GetRaw(), parameters.MTU, shared.BoolToEnabled(c.doIpConfig))

		ifconfig := c.getPlatformSpecifics(water.Config{
			DeviceType: mode.ToWaterDeviceType(),
		})
		c.iface, err = water.New(ifconfig)
		if err != nil {
			return err
		}

		log.Printf("[%s] Opened %s", c.socket.GetConnectionID(), c.iface.Name())

		c.setMTUNoInterface(parameters.MTU)
		err = c.configureInterface()
		if err != nil {
			return err
		}

		err = c.addRoute(c.remoteNet.GetSubnet())
		if err != nil {
			return err
		}

		log.Printf("[%s] Configured interface, starting operations", c.socket.GetConnectionID())
		err = c.socket.SetInterface(c.iface)
		if err != nil {
			return err
		}

		go c.runEventScript(clientEventUp)

		return nil
	})
}
