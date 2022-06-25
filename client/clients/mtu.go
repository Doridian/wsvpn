package clients

func (c *Client) setMTUNoInterface(mtu int) {
	c.mtu = mtu

	if c.socket != nil {
		c.socket.SetMTU(mtu)
	}
}

func (c *Client) SetMTU(mtu int) error {
	c.setMTUNoInterface(mtu)
	if c.iface != nil {
		return c.configureInterfaceMTU()
	}
	return nil
}
