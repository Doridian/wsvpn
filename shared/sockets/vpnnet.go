package sockets

import (
	"errors"

	"github.com/Doridian/wsvpn/shared"
	"github.com/Doridian/wsvpn/shared/iface"
)

func (s *Socket) SetInterface(iface *iface.WaterInterfaceWrapper) error {
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

func (s *Socket) GetInterfaceIfManaged() *iface.WaterInterfaceWrapper {
	if !s.ifaceManaged {
		return nil
	}
	return s.iface
}

func (s *Socket) tryServeIfaceRead() {
	if s.iface == nil || !s.ifaceManaged {
		return
	}

	s.wg.Add(1)
	go func() {
		defer s.closeDone()

		packet := make([]byte, 0)

		for {
			if len(packet) != s.packetBufferSize {
				packet = make([]byte, s.packetBufferSize)
			}

			n, err := s.iface.Interface.Read(packet)
			if err != nil {
				s.log.Printf("Error reading packet from tun: %v", err)
				return
			}

			if n < 1 || n >= len(packet) {
				continue
			}

			err = s.WritePacket(packet[:n])
			if err != nil {
				return
			}
		}
	}()
}
