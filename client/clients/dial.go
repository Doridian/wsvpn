package clients

import (
	"net"
	"syscall"
)

func (c *Client) DialerControlFunc(network, address string, conn syscall.RawConn) error {
	return setFirewallMarkRaw(conn, c.FirewallMark)
}

func (c *Client) GetDialer() *net.Dialer {
	return c.dialer
}
