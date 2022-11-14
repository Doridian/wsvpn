package macswitch

import (
	"time"

	"github.com/Doridian/water/waterutil"
	"github.com/Doridian/wsvpn/shared/sockets"
	lru "github.com/hashicorp/golang-lru/v2"
)

const EthernetLength = 14

func (g *MACSwitch) HandlePacket(socket *sockets.Socket, packet []byte) (bool, error) {
	if len(packet) < EthernetLength {
		return !g.AllowUnknownEtherTypes, nil
	}

	etherType := waterutil.MACEthertype(packet)
	if !g.AllowUnknownEtherTypes && etherType != waterutil.ARP && etherType != waterutil.IPv4 && etherType != waterutil.IPv6 {
		return true, nil
	}

	if socket != nil {
		if !g.setMACFrom(socket, packet) {
			return true, nil
		}

		if !g.AllowIPSpoofing {
			expectedIPVersion := byte(0)
			expectedMinLen := 0
			switch etherType {
			case waterutil.IPv4:
				expectedIPVersion = 4
				expectedMinLen = 20
			case waterutil.IPv6:
				expectedIPVersion = 6
				expectedMinLen = 40
			}

			if expectedIPVersion > 0 {
				if len(packet) < EthernetLength+expectedMinLen {
					return true, nil
				}

				if waterutil.IPVersion(packet[EthernetLength:]) != expectedIPVersion {
					return true, nil
				}

				srcIP := waterutil.IPSource(packet[EthernetLength:])
				if !srcIP.Equal(socket.AssignedIP) {
					return true, nil
				}
			}
		}
	}

	if socket == nil || g.AllowClientToClient {
		destMAC := waterutil.MACDestination(packet)

		if waterutil.IsMACUnicast(destMAC) {
			socketDest := g.findSocketByMAC(destMAC)
			if socketDest != nil {
				_ = socketDest.WritePacket(packet)
				return true, nil
			}
		} else {
			g.broadcastDataMessage(packet, socket)
		}
	}

	return false, nil
}

func (g *MACSwitch) onMACEvicted(key macAddr, value time.Time) {
	g.macLock.Lock()
	defer g.macLock.Unlock()

	delete(g.macTable, key)
}

func (g *MACSwitch) RegisterSocket(socket *sockets.Socket) {
	g.macLock.Lock()
	defer g.macLock.Unlock()

	var err error
	g.socketTable[socket], err = lru.NewWithEvict(g.AllowedMACsPerConnection, g.onMACEvicted)
	if err != nil {
		go socket.CloseError(err)
	}
}

func (g *MACSwitch) UnregisterSocket(socket *sockets.Socket) {
	g.macLock.Lock()

	socketTbl := g.socketTable[socket]
	if socketTbl == nil {
		g.macLock.Unlock()
		return
	}
	socketMACs := socketTbl.Keys()

	for _, mac := range socketMACs {
		delete(g.macTable, mac)
	}

	delete(g.socketTable, socket)

	g.macLock.Unlock()

	socketTbl.Purge()
}
