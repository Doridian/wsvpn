package iface

import (
	"fmt"
	"net"
)

func NetworkInterfaceExists(name string) bool {
	iface, err := net.InterfaceByName(name)
	if err != nil {
		return false
	}
	return iface != nil
}

func FindLowestNetworkInterfaceByPrefix(prefix string) string {
	i := 0
	var ifaceName string
	for {
		ifaceName = fmt.Sprintf("%s%d", prefix, i)
		if !NetworkInterfaceExists(ifaceName) {
			return ifaceName
		}
		i += 1
	}
}
