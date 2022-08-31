package macswitch

import (
	"github.com/Doridian/water/waterutil"
	"github.com/Doridian/wsvpn/shared/sockets"
	lru "github.com/hashicorp/golang-lru"
)

const ETH_LEN = 14

func (g *MACSwitch) HandlePacket(socket *sockets.Socket, packet []byte) (bool, error) {
	if len(packet) < ETH_LEN {
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

		if !g.AllowIpSpoofing {
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
				if len(packet) < ETH_LEN+expectedMinLen {
					return true, nil
				}

				if waterutil.IPVersion(packet[ETH_LEN:]) != expectedIPVersion {
					return true, nil
				}

				srcIp := waterutil.IPSource(packet[ETH_LEN:])
				if !srcIp.Equal(socket.AssignedIP) {
					return true, nil
				}
			}
		}
	}

	if socket == nil || g.AllowClientToClient {
		destMac := waterutil.MACDestination(packet)

		if waterutil.IsMACUnicast(destMac) {
			socketDest := g.findSocketByMAC(destMac)
			if socketDest != nil {
				socketDest.WritePacket(packet)
				return true, nil
			}
		} else {
			g.broadcastDataMessage(packet, socket)
		}
	}

	return false, nil
}

func (g *MACSwitch) onMACEvicted(key interface{}, value interface{}) {
	macAddr := key.(macAddr)

	g.macLock.Lock()
	defer g.macLock.Unlock()

	delete(g.macTable, macAddr)
}

func (g *MACSwitch) RegisterSocket(socket *sockets.Socket) {
	g.macLock.Lock()
	defer g.macLock.Unlock()

	var err error
	g.socketTable[socket], err = lru.NewWithEvict(g.AllowedMacsPerConnection, g.onMACEvicted)
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
	socketMacs := socketTbl.Keys()

	for _, mac := range socketMacs {
		delete(g.macTable, mac.(macAddr))
	}

	delete(g.socketTable, socket)

	g.macLock.Unlock()

	socketTbl.Purge()
}
