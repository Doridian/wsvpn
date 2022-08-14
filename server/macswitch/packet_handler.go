package macswitch

import (
	"github.com/Doridian/wsvpn/shared"
	"github.com/Doridian/wsvpn/shared/sockets"
	lru "github.com/hashicorp/golang-lru"
)

const ETH_LEN = 14

func (g *MACSwitch) HandlePacket(socket *sockets.Socket, packet []byte) (bool, error) {
	if len(packet) < ETH_LEN {
		return !g.AllowUnknownEtherTypes, nil
	}

	etherType := shared.GetEtherType(packet)
	if !g.AllowUnknownEtherTypes && etherType != shared.ETHTYPE_ARP && etherType != shared.ETHTYPE_IPV4 {
		return true, nil
	}

	if socket != nil {
		if !g.setMACFrom(socket, packet) {
			return true, nil
		}

		if !g.AllowIpSpoofing && etherType == shared.ETHTYPE_IPV4 {
			if len(packet) < ETH_LEN+20 {
				return !g.AllowUnknownEtherTypes, nil
			}

			srcIp := shared.GetSrcIPv4(packet, ETH_LEN)
			if srcIp != socket.AssignedIP {
				return true, nil
			}
		}
	}

	if socket == nil || g.AllowClientToClient {
		dest := shared.GetDestMAC(packet)

		if shared.MACIsUnicast(dest) {
			socket_dest := g.findSocketByMAC(dest)
			if socket_dest != nil {
				socket_dest.WritePacket(packet)
				return true, nil
			}
		} else {
			g.broadcastDataMessage(packet, socket)
		}
	}

	return false, nil
}

func (g *MACSwitch) onMacEvicted(key interface{}, value interface{}) {
	macAddr := key.(shared.MacAddr)

	g.macLock.Lock()
	defer g.macLock.Unlock()

	delete(g.macTable, macAddr)
}

func (g *MACSwitch) RegisterSocket(socket *sockets.Socket) {
	g.macLock.Lock()
	defer g.macLock.Unlock()

	var err error
	g.socketTable[socket], err = lru.NewWithEvict(g.AllowedMacsPerConnection, g.onMacEvicted)
	if err != nil {
		socket.CloseError(err)
	}
}

func (g *MACSwitch) UnregisterSocket(socket *sockets.Socket) {
	g.macLock.Lock()

	socketTbl := g.socketTable[socket]
	if socketTbl == nil {
		g.macLock.Unlock()
		return
	}
	socketMacs := socketTbl.Keys()

	for _, mac := range socketMacs {
		delete(g.macTable, mac.(shared.MacAddr))
	}

	delete(g.socketTable, socket)

	g.macLock.Unlock()

	socketTbl.Purge()
}
