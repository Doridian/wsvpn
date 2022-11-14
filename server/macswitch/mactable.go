package macswitch

import (
	"errors"
	"net"
	"time"

	"github.com/Doridian/water/waterutil"
	"github.com/Doridian/wsvpn/shared/sockets"
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
		_ = socket.WritePacket(data)
	}
}

func (g *MACSwitch) findSocketByMAC(hwAddr net.HardwareAddr) *sockets.Socket {
	mac := hwAddrToMAC(hwAddr)

	g.macLock.RLock()
	defer g.macLock.RUnlock()

	return g.macTable[mac]
}

func (g *MACSwitch) cleanupAllMACs() {
	for g.isRunning {
		<-g.cleanupTimer.C
		if !g.AllowMACChanging {
			continue
		}

		g.macLock.RLock()
		tables := make([]socketToMACs, 0, len(g.socketTable))
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

func (g *MACSwitch) cleanupMACs(macTable socketToMACs) {
	for {
		k, v, ok := macTable.GetOldest()
		if !ok || time.Since(v) <= g.MACTableTimeout {
			break
		}
		macTable.Remove(k)
	}
}

func (g *MACSwitch) setMACFrom(socket *sockets.Socket, msg []byte) bool {
	srcMAC := waterutil.MACSource(msg)
	if !waterutil.IsMACUnicast(srcMAC) {
		return false
	}

	srcMACAddr := hwAddrToMAC(srcMAC)

	g.macLock.RLock()
	socketMACs := g.socketTable[socket]
	g.macLock.RUnlock()

	if socketMACs == nil {
		return false
	}

	if socketMACs.Contains(srcMACAddr) {
		socketMACs.Add(srcMACAddr, time.Now())
		return true
	}

	if !g.AllowMACChanging && socketMACs.Len() > 0 {
		return false
	}

	g.macLock.Lock()
	if g.macTable[srcMACAddr] != nil {
		g.macLock.Unlock()
		socket.CloseError(errors.New("MAC collision"))
		return false
	}

	g.macTable[srcMACAddr] = socket
	g.macLock.Unlock()

	socketMACs.Add(srcMACAddr, time.Now())

	return true
}
