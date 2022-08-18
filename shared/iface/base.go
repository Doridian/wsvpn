package iface

import (
	"fmt"
	"log"
	"net"

	"github.com/Doridian/water"
	"github.com/Doridian/wsvpn/shared"
)

type WaterInterfaceWrapper struct {
	Interface *water.Interface
}

func NewInterfaceWrapper(iface *water.Interface) *WaterInterfaceWrapper {
	return &WaterInterfaceWrapper{
		Interface: iface,
	}
}

func (w *WaterInterfaceWrapper) Close() error {
	return w.Interface.Close()
}

func (w *WaterInterfaceWrapper) splitSubnet(ipNet *shared.VPNNet, ipLocal net.IP) (ipNetSize int, ipLocalCidr string) {
	ipNetSize = 32
	if ipNet != nil {
		ipNetSize = ipNet.GetSize()
	}
	ipLocalCidr = fmt.Sprintf("%s/%d", ipLocal.String(), ipNetSize)
	return
}

func (w *WaterInterfaceWrapper) addPeerRoute(ipNetSize int, ipPeer net.IP) error {
	if ipNetSize != 32 {
		return nil
	}
	err := w.AddInterfaceRoute(&net.IPNet{
		IP:   ipPeer,
		Mask: net.CIDRMask(32, 32),
	})

	if err != nil {
		log.Printf("Error adding peer route %s for %s: %v", ipPeer.String(), w.Interface.Name(), err)
	}

	return nil
}

func getInterfaceNameOrPrefix(ifaceConfig *InterfaceConfig) string {
	if ifaceConfig.OneInterfacePerConnection && ifaceConfig.Name != "" {
		return shared.FindLowestNetworkInterfaceByPrefix(ifaceConfig.Name)
	}

	return ifaceConfig.Name
}
