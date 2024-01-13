//go:build !linux

package clients

import (
	"net"
	"syscall"
)

func setFirewallMark(conn net.Conn, mark int) error {
	return nil
}

func setFirewallMarkRaw(fd syscall.RawConn, mark int) error {
	return nil
}
