package shared

import (
	"fmt"
)

const (
	EventUp      = "up"
	EventDown    = "down"
	EventStartup = "startup"
)

type EventConfigHolder struct {
	UpScript      []string
	DownScript    []string
	StartupScript []string
}

type EventConfig struct {
	Up      []string `yaml:"up"`
	Down    []string `yaml:"down"`
	Startup []string `yaml:"startup"`
}

func (c *EventConfigHolder) RunEventScript(op string, remoteNet string, iface string, args ...string) error {
	var script []string

	switch op {
	case EventUp:
		script = c.UpScript
	case EventDown:
		script = c.DownScript
	case EventStartup:
		script = c.StartupScript
	default:
		return fmt.Errorf("invalid event %s", op)
	}

	if len(script) == 0 {
		return nil
	}

	allArgs := []string{}
	if len(script) > 1 {
		allArgs = append(allArgs, script[1:]...)
	}
	allArgs = append(allArgs, op, remoteNet, iface)
	allArgs = append(allArgs, args...)
	return ExecCmd(script[0], allArgs...)
}

func (c *EventConfigHolder) LoadEventConfig(config *EventConfig) {
	c.UpScript = config.Up
	c.DownScript = config.Down
	c.StartupScript = config.Startup
}
