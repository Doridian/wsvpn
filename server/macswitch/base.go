package macswitch

import (
	"sync"
	"time"

	"github.com/Doridian/wsvpn/shared"
	"github.com/Doridian/wsvpn/shared/sockets"
	lru "github.com/hashicorp/golang-lru"
)

type socketToMacs = *lru.Cache

type MACSwitch struct {
	AllowClientToClient      bool
	AllowIpSpoofing          bool
	AllowUnknownEtherTypes   bool
	AllowMacChanging         bool
	AllowedMacsPerConnection int
	MacTableTimeout          time.Duration

	macTable     map[shared.MacAddr]*sockets.Socket
	socketTable  map[*sockets.Socket]socketToMacs
	macLock      *sync.RWMutex
	cleanupTimer *time.Timer
	isRunning    bool
}

func MakeMACSwitch() *MACSwitch {
	sw := &MACSwitch{
		AllowClientToClient:      false,
		AllowIpSpoofing:          false,
		AllowUnknownEtherTypes:   false,
		AllowMacChanging:         false,
		AllowedMacsPerConnection: 1,
		MacTableTimeout:          time.Duration(600 * time.Second),
		macTable:                 make(map[shared.MacAddr]*sockets.Socket),
		socketTable:              make(map[*sockets.Socket]socketToMacs),
		macLock:                  &sync.RWMutex{},
		cleanupTimer:             time.NewTimer(time.Duration(30 * time.Second)),
		isRunning:                true,
	}

	go sw.cleanupAllMACs()

	return sw
}

func (g *MACSwitch) Close() {
	g.isRunning = false
}
