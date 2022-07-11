package macswitch

import (
	"github.com/Doridian/wsvpn/shared"
	"github.com/Doridian/wsvpn/shared/sockets"
)

func (g *MACSwitch) HandlePacket(socket *sockets.Socket, packet []byte) (bool, error) {
	if len(packet) < 14 {
		return false, nil
	}

	if socket != nil {
		g.setMACFrom(socket, packet)
	}

	if socket == nil || g.AllowClientToClient {
		dest := shared.GetDestMAC(packet)

		if shared.MACIsUnicast(dest) {
			socket_dest := g.findSocketByMAC(dest)
			if socket_dest != nil {
				socket_dest.WritePacket(packet)
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

func (g *MACSwitch) RegisterSocket(socket *sockets.Socket) {

}

func (g *MACSwitch) UnregisterSocket(socket *sockets.Socket) {
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
