package clients

func (c *Client) SetMTU(mtu int) error {
	c.mtu = mtu
	if c.socket != nil {
		c.socket.SetMTU(mtu)
	}
	if c.iface != nil {
		return c.iface.SetMTU(mtu)
	}
	return nil
}
