package ipswitch

import (
	"errors"

	"github.com/Doridian/wsvpn/shared"
	"github.com/Doridian/wsvpn/shared/sockets"
)

const IP_LEN = 20

func (g *IPSwitch) HandlePacket(socket *sockets.Socket, packet []byte) (bool, error) {
	if len(packet) < IP_LEN {
		return true, nil
	}

	srcIp := shared.GetSrcIPv4(packet, 0)
	if srcIp != socket.AssignedIP {
		return true, nil
	}

	if socket == nil || g.AllowClientToClient {
		dest := shared.GetDestIPv4(packet, 0)

		if shared.IPv4IsUnicast(dest) {
			socket_dest := g.findSocketByIP(dest)
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

func (g *IPSwitch) RegisterSocket(socket *sockets.Socket) {
	g.ipLock.Lock()
	oldSocket, ok := g.ipTable[socket.AssignedIP]
	g.ipTable[socket.AssignedIP] = socket
	g.ipLock.Unlock()

	if ok {
		oldSocket.CloseError(errors.New("IP conflict"))
	}
}

func (g *IPSwitch) UnregisterSocket(socket *sockets.Socket) {
	g.ipLock.Lock()
	defer g.ipLock.Unlock()

	ourSocket, ok := g.ipTable[socket.AssignedIP]
	if !ok || ourSocket != socket {
		return
	}

	delete(g.ipTable, socket.AssignedIP)
}
