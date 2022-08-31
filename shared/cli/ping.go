package cli

import (
	"time"

	"github.com/Doridian/wsvpn/shared/sockets"
)

type PingConfig struct {
	Interval time.Duration `yaml:"interval"`
	Timeout  time.Duration `yaml:"timeout"`
}

type PingFlagsSocketConfigurator struct {
	Config *PingConfig
}

func MakeDefaultPingConfig() PingConfig {
	return PingConfig{
		Interval: time.Duration(30) * time.Second,
		Timeout:  time.Duration(5) * time.Second,
	}
}

func (c *PingFlagsSocketConfigurator) ConfigureSocket(sock *sockets.Socket) error {
	sock.ConfigurePing(c.Config.Interval, c.Config.Timeout)
	return nil
}
