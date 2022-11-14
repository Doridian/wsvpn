package macswitch

import (
	"net"
	"sync"
	"time"

	"github.com/Doridian/wsvpn/shared/sockets"
	lru "github.com/hashicorp/golang-lru/v2"
)

type socketToMACs = *lru.Cache[macAddr, time.Time]

type macAddr [6]byte

func hwAddrToMAC(hw net.HardwareAddr) macAddr {
	var out macAddr
	copy(out[:], hw)
	return out
}

type MACSwitch struct {
	AllowClientToClient      bool
	AllowIPSpoofing          bool
	AllowUnknownEtherTypes   bool
	AllowMACChanging         bool
	AllowedMACsPerConnection int
	MACTableTimeout          time.Duration

	macTable     map[macAddr]*sockets.Socket
	socketTable  map[*sockets.Socket]socketToMACs
	macLock      *sync.RWMutex
	cleanupTimer *time.Timer
	isRunning    bool
}

func MakeMACSwitch() *MACSwitch {
	sw := &MACSwitch{
		AllowClientToClient:      false,
		AllowIPSpoofing:          false,
		AllowUnknownEtherTypes:   false,
		AllowMACChanging:         true,
		AllowedMACsPerConnection: 1,
		MACTableTimeout:          time.Duration(600 * time.Second),
		macTable:                 make(map[macAddr]*sockets.Socket),
		socketTable:              make(map[*sockets.Socket]socketToMACs),
		macLock:                  &sync.RWMutex{},
		cleanupTimer:             time.NewTimer(time.Duration(30 * time.Second)),
		isRunning:                true,
	}

	go sw.cleanupAllMACs()

	return sw
}

func (g *MACSwitch) ConfigUpdate() {
	g.macLock.RLock()
	tables := make([]socketToMACs, 0, len(g.socketTable))
	for _, table := range g.socketTable {
		tables = append(tables, table)
	}
	g.macLock.RUnlock()

	for _, table := range tables {
		table.Resize(g.AllowedMACsPerConnection)
	}
}

func (g *MACSwitch) Close() {
	g.isRunning = false
}
