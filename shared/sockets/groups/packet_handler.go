package groups

import (
	"github.com/Doridian/wsvpn/shared"
	"github.com/Doridian/wsvpn/shared/sockets"
)

func (g *SocketGroup) HandlePacket(socket *sockets.Socket, packet []byte) (bool, error) {
	if len(packet) < 14 {
		return false, nil
	}

	if socket != nil {
		g.setMACFrom(socket, packet)
	}

	if socket == nil || g.AllowClientToClient {
		dest := shared.GetDestMAC(packet)

		if shared.MACIsUnicast(dest) {
			sd := g.findSocketByMAC(dest)
			if sd != nil {
				sd.WriteDataMessage(packet)
			} else {
				return false, nil
			}
		} else {
			g.broadcastDataMessage(packet, socket)
		}

		return true, nil
	}

	return false, nil
}

func (g *SocketGroup) RegisterSocket(socket *sockets.Socket) {

}

func (g *SocketGroup) UnregisterSocket(socket *sockets.Socket) {
	g.macLock.Lock()
	defer g.macLock.Unlock()
	socketMac, ok := g.socketTable[socket]
	if !ok {
		socketMac = shared.DefaultMac
	}

	if socketMac != shared.DefaultMac {
		delete(g.macTable, socketMac)
		delete(g.socketTable, socket)
	}
}
