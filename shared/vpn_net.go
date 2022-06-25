package shared

import (
	"net"

	"github.com/apparentlymart/go-cidr/cidr"
)

type VPNNet struct {
	ip    net.IP
	ipNet *net.IPNet
	str   string
}

func (r *VPNNet) GetRawIP() net.IP {
	return r.ip
}

func (r *VPNNet) GetServerIP() net.IP {
	ip, _ := r.GetIPAt(1)
	return ip
}

func (r *VPNNet) GetIPAt(idx int) (net.IP, error) {
	ip, err := cidr.Host(r.ipNet, idx)
	return ip, err
}

func (r *VPNNet) GetNetmask() string {
	return IPNetGetNetMask(r.ipNet)
}

func (r *VPNNet) GetSubnet() *net.IPNet {
	return r.ipNet
}

func (r *VPNNet) GetRaw() string {
	return r.str
}

func (r *VPNNet) GetSize() int {
	subnetOnes, _ := r.ipNet.Mask.Size()
	return subnetOnes
}

func (r *VPNNet) GetClientSlots() uint64 {
	return cidr.AddressCount(r.ipNet) - 3
}

func ParseVPNNet(rNetStr string) (*VPNNet, error) {
	ip, ipNet, err := net.ParseCIDR(rNetStr)
	if err != nil {
		return nil, err
	}
	return &VPNNet{
		ip:    ip,
		ipNet: ipNet,
		str:   rNetStr,
	}, nil
}
