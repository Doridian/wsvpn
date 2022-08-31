package shared

import "github.com/Doridian/water"

type VPNMode int

const (
	VPNModeTUN VPNMode = iota
	VPNModeTAP
	VPNModeInvalid
)

func (m VPNMode) ToString() string {
	switch m {
	case VPNModeTAP:
		return "TAP"
	case VPNModeTUN:
		return "TUN"
	}
	return "Invalid"
}

func (m VPNMode) ToWaterDeviceType() water.DeviceType {
	switch m {
	case VPNModeTAP:
		return water.TAP
	case VPNModeTUN:
		return water.TUN
	}
	return -1
}

func VPNModeFromString(mode string) VPNMode {
	switch mode {
	case "TAP":
		return VPNModeTAP
	case "TUN":
		return VPNModeTUN
	}
	return VPNModeInvalid
}
