package main

import (
	"fmt"
	"net"

	"github.com/apparentlymart/go-cidr/cidr"
)

type remoteNet struct {
	ip    net.IP
	ipNet *net.IPNet
	str   string
}

func (r *remoteNet) getClientIP() string {
	return r.ip.String()
}

func (r *remoteNet) getServerIP() string {
	ip, _ := cidr.Host(r.ipNet, 1)
	return ip.String()
}

func (r *remoteNet) getNetmask() string {
	mask := r.ipNet.Mask
	return fmt.Sprintf("%d.%d.%d.%d", mask[0], mask[1], mask[2], mask[3])
}

func parseRemoteNet(rNetStr string) (*remoteNet, error) {
	ip, ipNet, err := net.ParseCIDR(rNetStr)
	if err != nil {
		return nil, err
	}
	return &remoteNet{
		ip:    ip,
		ipNet: ipNet,
		str:   rNetStr,
	}, nil
}
