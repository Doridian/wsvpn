package sockets

import (
	"errors"
	"log"

	"github.com/Doridian/wsvpn/shared"
	"github.com/songgao/water"
)

func (s *Socket) SetInterface(iface *water.Interface) error {
	if s.iface != nil {
		return errors.New("cannot re-define interface: Already set")
	}
	s.iface = iface
	s.tryServeIfaceRead()
	return nil
}

func (s *Socket) SetMTU(mtu int) {
	s.packetBufferSize = shared.GetPacketBufferSizeByMTU(mtu)
}

func (s *Socket) tryServeIfaceRead() {
	if s.iface == nil || !s.ifaceManaged {
		return
	}

	s.wg.Add(1)
	go func() {
		defer s.closeDone()

		packet := make([]byte, s.packetBufferSize)

		for {
			n, err := s.iface.Read(packet)
			if err != nil {
				log.Printf("[%s] Error reading packet from tun: %v", s.ConnectionID, err)
				return
			}

			err = s.WriteDataMessage(packet[:n])
			if err != nil {
				return
			}
		}
	}()
}
