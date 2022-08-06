package macswitch

import (
	"log"

	"github.com/Doridian/wsvpn/shared"
	"github.com/Doridian/wsvpn/shared/sockets"
)

const ETH_LEN = 14

func (g *MACSwitch) HandlePacket(socket *sockets.Socket, packet []byte) (bool, error) {
	if len(packet) < ETH_LEN {
		return !g.AllowUnknownEtherTypes, nil
	}

	etherType := shared.GetEtherType(packet)
	if !g.AllowUnknownEtherTypes && etherType != shared.ETHTYPE_ARP && etherType != shared.ETHTYPE_IPV4 {
		log.Printf("Unknown EtherType: %d", etherType)
		return true, nil
	}

	if socket != nil {
		g.setMACFrom(socket, packet)

		if !g.AllowIpSpoofing && etherType == shared.ETHTYPE_IPV4 {
			if len(packet) < ETH_LEN+20 {
				log.Printf("TooShort v4: %d", len(packet))
				return !g.AllowUnknownEtherTypes, nil
			}

			srcIp := shared.GetSrcIPv4(packet, ETH_LEN)
			if srcIp != socket.AssignedIP {
				log.Printf("WrongIP: %v vs %v", srcIp, socket.AssignedIP)
				return true, nil
			}
		}
	}

	if socket == nil || g.AllowClientToClient {
		dest := shared.GetDestMAC(packet)

		if shared.MACIsUnicast(dest) {
			socket_dest := g.findSocketByMAC(dest)
			if socket_dest != nil {
				log.Printf("HHU: %d", etherType)
				socket_dest.WritePacket(packet)
			} else {
				log.Printf("UNH: %d", etherType)
				return false, nil
			}
		} else {
			log.Printf("BCM: %d", etherType)
			g.broadcastDataMessage(packet, socket)
		}

		log.Printf("HHM: %d", etherType)
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
