package groups

import (
	"log"

	"github.com/Doridian/wsvpn/shared"
	"github.com/Doridian/wsvpn/shared/sockets"
)

func (g *SocketGroup) BroadcastDataMessage(data []byte, skip *sockets.Socket) {
	g.macLock.RLock()
	targetList := make([]*sockets.Socket, 0, len(g.macTable))
	for _, v := range g.macTable {
		if v == skip {
			continue
		}
		targetList = append(targetList, v)
	}
	g.macLock.RUnlock()

	for _, v := range targetList {
		v.WriteDataMessage(data)
	}
}

func (g *SocketGroup) findSocketByMAC(mac shared.MacAddr) *sockets.Socket {
	g.macLock.RLock()
	defer g.macLock.RUnlock()

	return g.macTable[mac]
}

func (g *SocketGroup) setMACFrom(socket *sockets.Socket, msg []byte) {
	srcMac := shared.GetSrcMAC(msg)
	socketMac, ok := g.socketTable[socket]
	if !ok {
		socketMac = shared.DefaultMac
	}

	if !shared.MACIsUnicast(srcMac) || srcMac == socketMac {
		return
	}

	g.macLock.Lock()
	defer g.macLock.Unlock()
	if socketMac != shared.DefaultMac {
		delete(g.macTable, socketMac)
		delete(g.socketTable, socket)
	}

	if g.macTable[srcMac] != nil {
		log.Printf("[%s] MAC collision: Killing", socket.GetConnectionID())
		socket.Close()
		return
	}

	g.socketTable[socket] = srcMac
	g.macTable[srcMac] = socket
}
