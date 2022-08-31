package shared

import "github.com/Doridian/water"

type VPNMode int

const (
	VPN_MODE_TUN VPNMode = iota
	VPN_MODE_TAP
	VPN_MODE_INVALID
)

func (m VPNMode) ToString() string {
	switch m {
	case VPN_MODE_TAP:
		return "TAP"
	case VPN_MODE_TUN:
		return "TUN"
	}
	return "Invalid"
}

func (m VPNMode) ToWaterDeviceType() water.DeviceType {
	switch m {
	case VPN_MODE_TAP:
		return water.TAP
	case VPN_MODE_TUN:
		return water.TUN
	}
	return -1
}

func VPNModeFromString(mode string) VPNMode {
	switch mode {
	case "TAP":
		return VPN_MODE_TAP
	case "TUN":
		return VPN_MODE_TUN
	}
	return VPN_MODE_INVALID
}
