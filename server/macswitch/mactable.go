package macswitch

import (
	"errors"
	"net"
	"time"

	"github.com/Doridian/water/waterutil"
	"github.com/Doridian/wsvpn/shared/sockets"
	lru "github.com/hashicorp/golang-lru"
)

func (g *MACSwitch) broadcastDataMessage(data []byte, skip *sockets.Socket) {
	g.macLock.RLock()
	targetList := make([]*sockets.Socket, 0, len(g.socketTable))
	for sock := range g.socketTable {
		if sock == skip {
			continue
		}
		targetList = append(targetList, sock)
	}
	g.macLock.RUnlock()

	for _, socket := range targetList {
		socket.WritePacket(data)
	}
}

func (g *MACSwitch) findSocketByMAC(hwAddr net.HardwareAddr) *sockets.Socket {
	mac := hwAddrToMac(hwAddr)

	g.macLock.RLock()
	defer g.macLock.RUnlock()

	return g.macTable[mac]
}

func (g *MACSwitch) cleanupAllMACs() {
	for g.isRunning {
		<-g.cleanupTimer.C
		if !g.AllowMacChanging {
			continue
		}

		g.macLock.RLock()
		tables := make([]*lru.Cache, 0, len(g.socketTable))
		for _, table := range g.socketTable {
			tables = append(tables, table)
		}
		g.macLock.RUnlock()

		for _, table := range tables {
			g.cleanupMACs(table)
		}
	}
	g.cleanupTimer.Stop()
}

func (g *MACSwitch) cleanupMACs(macTable *lru.Cache) {
	for {
		k, v, ok := macTable.GetOldest()
		if !ok || time.Since(v.(time.Time)) <= g.MacTableTimeout {
			break
		}
		macTable.Remove(k)
	}
}

func (g *MACSwitch) setMACFrom(socket *sockets.Socket, msg []byte) bool {
	srcMac := waterutil.MACSource(msg)
	if !waterutil.IsMACUnicast(srcMac) {
		return false
	}

	srcMacAddr := hwAddrToMac(srcMac)

	g.macLock.RLock()
	socketMacs := g.socketTable[socket]
	g.macLock.RUnlock()

	if socketMacs == nil {
		return false
	}

	if socketMacs.Contains(srcMacAddr) {
		socketMacs.Add(srcMacAddr, time.Now())
		return true
	}

	if !g.AllowMacChanging && socketMacs.Len() > 0 {
		return false
	}

	g.macLock.Lock()
	if g.macTable[srcMacAddr] != nil {
		g.macLock.Unlock()
		socket.CloseError(errors.New("MAC collision"))
		return false
	}

	g.macTable[srcMacAddr] = socket
	g.macLock.Unlock()

	socketMacs.Add(srcMacAddr, time.Now())

	return true
}
