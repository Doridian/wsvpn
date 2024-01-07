//go:build !linux

package clients

import (
	"net"
)

func setFirewallMark(conn net.Conn, mark int) error {
	return nil
}
