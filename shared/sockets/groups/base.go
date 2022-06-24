package groups

import (
	"sync"

	"github.com/Doridian/wsvpn/shared"
	"github.com/Doridian/wsvpn/shared/sockets"
)

type SocketGroup struct {
	AllowClientToClient bool

	macTable    map[shared.MacAddr]*sockets.Socket
	socketTable map[*sockets.Socket]shared.MacAddr
	macLock     sync.RWMutex
}

func MakeSocketGroup() *SocketGroup {
	return &SocketGroup{
		AllowClientToClient: false,
		macTable:            make(map[shared.MacAddr]*sockets.Socket),
	}
}
