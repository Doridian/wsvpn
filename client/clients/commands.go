package clients

import (
	"errors"
	"net"

	"github.com/Doridian/wsvpn/shared"
	"github.com/Doridian/wsvpn/shared/commands"
	"github.com/songgao/water"
)

func (c *Client) registerCommandHandlers() {
	c.socket.AddCommandHandler(commands.AddRouteCommandName, func(command *commands.IncomingCommand) error {
		var err error
		var parameters commands.AddRouteParameters
		err = command.DeserializeParameters(&parameters)
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
		err = command.DeserializeParameters(&parameters)
		if err != nil {
			return err
		}

		if parameters.ClientID != "" {
			c.clientID = parameters.ClientID
			shared.UpdateLogger(c.log, "CLIENT", c.clientID)
		}
		if parameters.ServerID != "" {
			c.serverID = parameters.ServerID
		}

		c.socket.SetAllowEnableFragmentation(parameters.EnableFragmentation)

		mode := shared.VPNModeFromString(parameters.Mode)

		c.remoteNet, err = shared.ParseVPNNet(parameters.IpAddress)
		if err != nil {
			return err
		}

		c.doIpConfig = parameters.DoIpConfig

		c.log.Printf("Network mode %s, Subnet %s, MTU %d, IPConfig %s", parameters.Mode, c.remoteNet.GetRaw(), parameters.MTU, shared.BoolToEnabled(c.doIpConfig))

		ifconfig := water.Config{
			DeviceType: mode.ToWaterDeviceType(),
		}

		err = c.getPlatformSpecifics(&ifconfig, c.InterfaceConfig)
		if err != nil {
			return err
		}

		c.iface, err = water.New(ifconfig)
		if err != nil {
			return err
		}

		c.log.Printf("Opened %s", c.iface.Name())

		c.setMTUNoInterface(parameters.MTU)
		err = c.configureInterface()
		if err != nil {
			return err
		}

		err = c.addRoute(c.remoteNet.GetSubnet())
		if err != nil {
			return err
		}

		c.log.Printf("Configured interface, starting operations")
		err = c.socket.SetInterface(c.iface)
		if err != nil {
			return err
		}

		c.doRunEventScript(shared.EventUp)
		c.sentUpEvent = true

		return nil
	})

	c.socket.AddCommandHandler(commands.SetMtuCommandName, func(command *commands.IncomingCommand) error {
		var err error
		var parameters commands.SetMtuParameters
		err = command.DeserializeParameters(&parameters)
		if err != nil {
			return err
		}

		c.log.Printf("Server requested MTU change to %d", parameters.MTU)
		return c.SetMTU(parameters.MTU)
	})
}
