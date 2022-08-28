package macswitch

import (
	"net"
	"sync"
	"time"

	"github.com/Doridian/wsvpn/shared/sockets"
	lru "github.com/hashicorp/golang-lru"
)

type socketToMacs = *lru.Cache

type macAddr [6]byte

func hwAddrToMac(hw net.HardwareAddr) macAddr {
	var out macAddr
	copy(out[:], hw)
	return out
}

func hwAddrIsUnicast(hw net.HardwareAddr) bool {
	return (hw[0] & 0b00000001) == 0
}

type MACSwitch struct {
	AllowClientToClient      bool
	AllowIpSpoofing          bool
	AllowUnknownEtherTypes   bool
	AllowMacChanging         bool
	AllowedMacsPerConnection int
	MacTableTimeout          time.Duration

	macTable     map[macAddr]*sockets.Socket
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
		AllowMacChanging:         true,
		AllowedMacsPerConnection: 1,
		MacTableTimeout:          time.Duration(600 * time.Second),
		macTable:                 make(map[macAddr]*sockets.Socket),
		socketTable:              make(map[*sockets.Socket]socketToMacs),
		macLock:                  &sync.RWMutex{},
		cleanupTimer:             time.NewTimer(time.Duration(30 * time.Second)),
		isRunning:                true,
	}

	go sw.cleanupAllMACs()

	return sw
}

func (g *MACSwitch) ConfigUpdate() {
	g.macLock.RLock()
	tables := make([]*lru.Cache, 0, len(g.socketTable))
	for _, table := range g.socketTable {
		tables = append(tables, table)
	}
	g.macLock.RUnlock()

	for _, table := range tables {
		table.Resize(g.AllowedMacsPerConnection)
	}
}

func (g *MACSwitch) Close() {
	g.isRunning = false
}
