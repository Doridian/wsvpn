package shared

import (
	"fmt"
)

const (
	EventUp   = "up"
	EventDown = "down"
)

type EventConfigHolder struct {
	UpScript   string
	DownScript string
}

type EventConfig struct {
	Up   string `yaml:"up"`
	Down string `yaml:"down"`
}

func (c *EventConfigHolder) RunEventScript(op string, remoteNet string, iface string) error {
	script := ""

	switch op {
	case EventUp:
		script = c.UpScript
	case EventDown:
		script = c.DownScript
	default:
		return fmt.Errorf("invalid event %s", op)
	}

	if script == "" {
		return nil
	}

	return ExecCmd(script, op, remoteNet, iface)
}

func (c *EventConfigHolder) LoadEventConfig(config *EventConfig) {
	c.UpScript = config.Up
	c.DownScript = config.Down
}
