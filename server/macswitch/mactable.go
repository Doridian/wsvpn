package macswitch

import (
	"errors"
	"time"

	"github.com/Doridian/wsvpn/shared"
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
		socket.WritePacket(data)
	}
}

func (g *MACSwitch) findSocketByMAC(mac shared.MacAddr) *sockets.Socket {
	g.macLock.RLock()
	defer g.macLock.RUnlock()

	return g.macTable[mac]
}

func (g *MACSwitch) cleanupAllMACs() {
	for g.isRunning {
		<-g.cleanupTimer.C
		for socket := range g.socketTable {
			g.cleanupMACs(socket)
		}
	}
	g.cleanupTimer.Stop()
}

func (g *MACSwitch) cleanupMACs(socket *sockets.Socket) {
	g.macLock.Lock()
	macTable := g.socketTable[socket]
	g.macLock.Unlock()
	if macTable == nil {
		return
	}

	for {
		k, v, ok := macTable.GetOldest()
		if !ok {
			break
		}
		if time.Since(v.(time.Time)) > g.MacTableTimeout {
			macTable.Remove(k)
		}
	}
}

func (g *MACSwitch) setMACFrom(socket *sockets.Socket, msg []byte) bool {
	srcMac := shared.GetSrcMAC(msg)
	socketMacs := g.socketTable[socket]

	if !shared.MACIsUnicast(srcMac) {
		return true
	}

	if socketMacs.Contains(srcMac) {
		socketMacs.Add(srcMac, time.Now())
		return true
	}

	if !g.AllowMacChanging && socketMacs.Len() > 0 {
		return false
	}

	g.macLock.Lock()
	if g.macTable[srcMac] != nil {
		g.macLock.Unlock()
		socket.CloseError(errors.New("MAC collision"))
		return false
	}

	g.macTable[srcMac] = socket
	g.macLock.Unlock()

	socketMacs.Add(srcMac, time.Now())

	return true
}
