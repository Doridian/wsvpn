package iface

import (
	"fmt"
	"log"
	"net"

	"github.com/Doridian/water"
	"github.com/Doridian/wsvpn/shared"
)

type WaterInterfaceWrapper struct {
	Interface    *water.Interface
	netInterface *net.Interface
}

func NewInterfaceWrapper(iface *water.Interface) *WaterInterfaceWrapper {
	return &WaterInterfaceWrapper{
		Interface:    iface,
		netInterface: nil,
	}
}

func (w *WaterInterfaceWrapper) Close() error {
	w.netInterface = nil
	return w.Interface.Close()
}

func (w *WaterInterfaceWrapper) getNetInterface() (*net.Interface, error) {
	if w.netInterface != nil {
		return w.netInterface, nil
	}

	iface, err := net.InterfaceByName(w.Interface.Name())
	if err != nil {
		return nil, err
	}
	w.netInterface = iface
	return iface, nil
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
		return FindLowestNetworkInterfaceByPrefix(ifaceConfig.Name)
	}

	return ifaceConfig.Name
}
