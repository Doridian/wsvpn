package ipswitch

import (
	"errors"

	"github.com/Doridian/water/waterutil"
	"github.com/Doridian/wsvpn/shared/sockets"
)

func (g *IPSwitch) HandlePacket(socket *sockets.Socket, packet []byte) (bool, error) {
	if len(packet) < 1 {
		return true, nil
	}

	expectedMinLen := 0
	switch waterutil.IPVersion(packet) {
	case 4:
		expectedMinLen = 20
	case 6:
		expectedMinLen = 40
	}

	if expectedMinLen < 1 || len(packet) < expectedMinLen {
		return true, nil
	}

	srcIp := waterutil.IPSource(packet)
	destIp := waterutil.IPDestination(packet)

	if srcIp == nil || destIp == nil {
		return true, nil
	}

	if socket != nil && !srcIp.Equal(socket.AssignedIP) {
		return true, nil
	}

	if socket == nil || g.AllowClientToClient {
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
	ipAddr := ipToIPAddr(socket.AssignedIP)

	g.ipLock.Lock()
	oldSocket, ok := g.ipTable[ipAddr]
	g.ipTable[ipAddr] = socket
	g.ipLock.Unlock()

	if ok {
		oldSocket.CloseError(errors.New("IP conflict"))
	}
}

func (g *IPSwitch) UnregisterSocket(socket *sockets.Socket) {
	ipAddr := ipToIPAddr(socket.AssignedIP)

	g.ipLock.Lock()
	defer g.ipLock.Unlock()

	ourSocket, ok := g.ipTable[ipAddr]
	if !ok || ourSocket != socket {
		return
	}

	delete(g.ipTable, ipAddr)
}
