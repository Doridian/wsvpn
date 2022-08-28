package ipswitch

import (
	"net"
	"sync"

	"github.com/Doridian/wsvpn/shared/sockets"
)

type ipv4 [4]byte

func ipToIPv4(ip net.IP) ipv4 {
	var out ipv4
	copy(out[:], ip.To4())
	return out
}

type IPSwitch struct {
	AllowClientToClient bool

	ipTable map[ipv4]*sockets.Socket
	ipLock  *sync.RWMutex
}

func MakeIPSwitch() *IPSwitch {
	return &IPSwitch{
		AllowClientToClient: false,
		ipTable:             make(map[ipv4]*sockets.Socket),
		ipLock:              &sync.RWMutex{},
	}
}
