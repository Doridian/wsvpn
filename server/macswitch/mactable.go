package macswitch

import (
	"errors"

	"github.com/Doridian/wsvpn/shared"
	"github.com/Doridian/wsvpn/shared/sockets"
)

func (g *MACSwitch) broadcastDataMessage(data []byte, skip *sockets.Socket) {
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

func (g *MACSwitch) findSocketByMAC(mac shared.MacAddr) *sockets.Socket {
	g.macLock.RLock()
	defer g.macLock.RUnlock()

	return g.macTable[mac]
}

func (g *MACSwitch) setMACFrom(socket *sockets.Socket, msg []byte) {
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
		socket.CloseError(errors.New("MAC collision"))
		return
	}

	g.socketTable[socket] = srcMac
	g.macTable[srcMac] = socket
}
