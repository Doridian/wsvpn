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

func (c *EventConfigHolder) RunEventScript(op string, remoteNet string, iface string, args ...string) error {
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

	all_args := []string{op, remoteNet, iface}
	all_args = append(all_args, args...)
	return ExecCmd(script, all_args...)
}

func (c *EventConfigHolder) LoadEventConfig(config *EventConfig) {
	c.UpScript = config.Up
	c.DownScript = config.Down
}
