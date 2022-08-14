package macswitch

import (
	"sync"

	"github.com/Doridian/wsvpn/shared"
	"github.com/Doridian/wsvpn/shared/sockets"
)

type MACSwitch struct {
	AllowClientToClient            bool
	AllowIpSpoofing                bool
	AllowUnknownEtherTypes         bool
	AllowMacChanging               bool
	AllowMultipleMacsPerConnection bool

	macTable    map[shared.MacAddr]*sockets.Socket
	socketTable map[*sockets.Socket]shared.MacAddr
	macLock     *sync.RWMutex
}

func MakeMACSwitch() *MACSwitch {
	return &MACSwitch{
		AllowClientToClient:            false,
		AllowIpSpoofing:                false,
		AllowUnknownEtherTypes:         false,
		AllowMacChanging:               false,
		AllowMultipleMacsPerConnection: false,
		macTable:                       make(map[shared.MacAddr]*sockets.Socket),
		socketTable:                    make(map[*sockets.Socket]shared.MacAddr),
		macLock:                        &sync.RWMutex{},
	}
}
