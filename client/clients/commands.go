package clients

import (
	"errors"
	"net"

	"github.com/Doridian/water"
	"github.com/Doridian/wsvpn/shared"
	"github.com/Doridian/wsvpn/shared/commands"
	"github.com/Doridian/wsvpn/shared/iface"
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

		err = c.iface.AddIPRoute(routeNet, c.remoteNet.GetServerIP())
		if err != nil {
			c.log.Printf("Error adding subnet route (not fatal): %v", err)
		}
		return nil
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

		c.socket.HandleInitPacketFragmentation(parameters.EnableFragmentation)

		mode := shared.VPNModeFromString(parameters.Mode)

		c.remoteNet, err = shared.ParseVPNNet(parameters.IPAddress)
		if err != nil {
			return err
		}

		c.doIPConfig = parameters.DoIPConfig

		c.socket.AssignedIP = c.remoteNet.GetRawIP()

		c.log.Printf("Network mode %s, Subnet %s, MTU %d, IPConfig %s", parameters.Mode, c.remoteNet.GetRaw(), parameters.MTU, shared.BoolToEnabled(c.doIPConfig))

		ifconfig := water.Config{
			DeviceType: mode.ToWaterDeviceType(),
		}

		err = iface.GetPlatformSpecifics(&ifconfig, c.InterfaceConfig)
		if err != nil {
			return err
		}

		localIface, err := water.New(ifconfig)
		if err != nil {
			return err
		}

		c.iface = iface.NewInterfaceWrapper(localIface)

		c.log.Printf("Opened interface %s", c.iface.Interface.Name())

		if c.doIPConfig {
			err = c.iface.Configure(c.remoteNet.GetRawIP(), c.remoteNet, c.remoteNet.GetServerIP())
		} else {
			err = c.iface.Configure(nil, c.remoteNet, c.remoteNet.GetServerIP())
		}
		if err != nil {
			return err
		}

		err = c.SetMTU(parameters.MTU)
		if err != nil {
			return err
		}

		err = c.socket.SetInterface(c.iface)
		if err != nil {
			return err
		}

		c.doRunEventScript(shared.EventUp)
		c.sentUpEvent = true

		c.log.Printf("Configured interface, VPN online")

		return nil
	})

	c.socket.AddCommandHandler(commands.SetMTUCommandName, func(command *commands.IncomingCommand) error {
		var err error
		var parameters commands.SetMTUParameters
		err = command.DeserializeParameters(&parameters)
		if err != nil {
			return err
		}

		c.log.Printf("Server requested MTU change to %d", parameters.MTU)
		return c.SetMTU(parameters.MTU)
	})
}
