package main

import (
	"net"

	"github.com/Doridian/wsvpn/shared"
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
	return shared.IPNetGetNetMask(r.ipNet)
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
