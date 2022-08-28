package ipswitch

import (
	"errors"
	"net"

	"github.com/Doridian/water/waterutil"
	"github.com/Doridian/wsvpn/shared/sockets"
)

const IP_LEN = 20

func (g *IPSwitch) HandlePacket(socket *sockets.Socket, packet []byte) (bool, error) {
	if len(packet) < IP_LEN {
		return true, nil
	}

	isIPv4 := waterutil.IsIPv4(packet)
	isIPv6 := waterutil.IsIPv6(packet)
	if !isIPv4 && !isIPv6 {
		return true, nil
	}

	var srcIp net.IP
	var destIp net.IP

	if isIPv4 {
		srcIp = waterutil.IPv4Source(packet)
		destIp = waterutil.IPv4Destination(packet)
	} else if isIPv6 {
		srcIp = packet[8:24]
		destIp = packet[24:40]
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
