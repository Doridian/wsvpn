package clients

import (
	"fmt"

	"github.com/Doridian/wsvpn/shared"
)

func (c *Client) runEventScript(op string) error {
	script := ""

	switch op {
	case clientEventUp:
		script = c.UpScript
	case clientEventDown:
		script = c.DownScript
	default:
		return fmt.Errorf("invalid event %s", op)
	}

	if script == "" {
		return nil
	}

	return shared.ExecCmd(script, op, c.remoteNet.GetRaw(), c.iface.Name())
}
