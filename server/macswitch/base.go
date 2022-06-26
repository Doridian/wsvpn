package macswitch

import (
	"sync"

	"github.com/Doridian/wsvpn/shared"
	"github.com/Doridian/wsvpn/shared/sockets"
)

type MACSwitch struct {
	AllowClientToClient bool

	macTable    map[shared.MacAddr]*sockets.Socket
	socketTable map[*sockets.Socket]shared.MacAddr
	macLock     *sync.RWMutex
}

func MakeMACSwitch() *MACSwitch {
	return &MACSwitch{
		AllowClientToClient: false,
		macTable:            make(map[shared.MacAddr]*sockets.Socket),
		socketTable:         make(map[*sockets.Socket]shared.MacAddr),
		macLock:             &sync.RWMutex{},
	}
}
