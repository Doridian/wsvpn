package ipswitch

import (
	"sync"

	"github.com/Doridian/wsvpn/shared"
	"github.com/Doridian/wsvpn/shared/sockets"
)

type IPSwitch struct {
	AllowClientToClient bool

	ipTable map[shared.IPv4]*sockets.Socket
	ipLock  *sync.RWMutex
}

func MakeIPSwitch() *IPSwitch {
	return &IPSwitch{
		AllowClientToClient: false,
		ipTable:             make(map[shared.IPv4]*sockets.Socket),
		ipLock:              &sync.RWMutex{},
	}
}
