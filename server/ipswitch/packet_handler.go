package ipswitch

import (
	"errors"

	"github.com/Doridian/water/waterutil"
	"github.com/Doridian/wsvpn/shared/sockets"
)

const IP_LEN = 20

func (g *IPSwitch) HandlePacket(socket *sockets.Socket, packet []byte) (bool, error) {
	if len(packet) < IP_LEN {
		return true, nil
	}

	if socket != nil {
		srcIp := waterutil.IPv4Source(packet)
		if !srcIp.Equal(socket.AssignedIP) {
			return true, nil
		}
	}

	if socket == nil || g.AllowClientToClient {
		destIp := waterutil.IPv4Destination(packet)

		if destIp.IsGlobalUnicast() {
			socketDest := g.findSocketByIP(destIp)
			if socketDest != nil {
				socketDest.WritePacket(packet)
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
	ip4 := ipToIPv4(socket.AssignedIP)

	g.ipLock.Lock()
	oldSocket, ok := g.ipTable[ip4]
	g.ipTable[ip4] = socket
	g.ipLock.Unlock()

	if ok {
		oldSocket.CloseError(errors.New("IP conflict"))
	}
}

func (g *IPSwitch) UnregisterSocket(socket *sockets.Socket) {
	ip4 := ipToIPv4(socket.AssignedIP)

	g.ipLock.Lock()
	defer g.ipLock.Unlock()

	ourSocket, ok := g.ipTable[ip4]
	if !ok || ourSocket != socket {
		return
	}

	delete(g.ipTable, ip4)
}
