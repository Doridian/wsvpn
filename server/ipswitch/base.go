package ipswitch

import (
	"net"
	"sync"

	"github.com/Doridian/wsvpn/shared/sockets"
)

type ipaddr [net.IPv6len]byte

func ipToIPAddr(ip net.IP) ipaddr {
	var out ipaddr
	copy(out[:], ip.To16())
	return out
}

type IPSwitch struct {
	AllowClientToClient bool

	ipTable map[ipaddr]*sockets.Socket
	ipLock  *sync.RWMutex
}

func MakeIPSwitch() *IPSwitch {
	return &IPSwitch{
		AllowClientToClient: false,
		ipTable:             make(map[ipaddr]*sockets.Socket),
		ipLock:              &sync.RWMutex{},
	}
}
