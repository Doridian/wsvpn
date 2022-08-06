package ipswitch

import (
	"github.com/Doridian/wsvpn/shared"
	"github.com/Doridian/wsvpn/shared/sockets"
)

func (g *IPSwitch) broadcastDataMessage(data []byte, skip *sockets.Socket) {
	g.ipLock.RLock()
	targetList := make([]*sockets.Socket, 0, len(g.ipTable))
	for _, v := range g.ipTable {
		if v == skip {
			continue
		}
		targetList = append(targetList, v)
	}
	g.ipLock.RUnlock()

	for _, socket := range targetList {
		socket.WritePacket(data)
	}
}

func (g *IPSwitch) findSocketByIP(ip shared.IPv4) *sockets.Socket {
	g.ipLock.RLock()
	defer g.ipLock.RUnlock()

	return g.ipTable[ip]
}
