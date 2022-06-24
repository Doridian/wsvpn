package sockets

import (
	"log"
	"sync"

	"github.com/Doridian/wsvpn/shared"
	"github.com/Doridian/wsvpn/shared/commands"
	"github.com/Doridian/wsvpn/shared/sockets/adapters"
	"github.com/songgao/water"
)

type Socket struct {
	connId                string
	adapter               adapters.SocketAdapter
	iface                 *water.Interface
	ifaceManaged          bool
	wg                    *sync.WaitGroup
	handlers              map[string]CommandHandler
	closechan             chan bool
	closechanopen         bool
	mac                   shared.MacAddr
	remoteProtocolVersion int
	packetBufferSize      int
	packetHandler         PacketHandler
}

func MakeSocket(connId string, adapter adapters.SocketAdapter, iface *water.Interface, ifaceManaged bool) *Socket {
	return &Socket{
		connId:                connId,
		adapter:               adapter,
		iface:                 iface,
		ifaceManaged:          ifaceManaged,
		wg:                    &sync.WaitGroup{},
		handlers:              make(map[commands.CommandName]CommandHandler),
		closechan:             make(chan bool),
		closechanopen:         true,
		mac:                   shared.DefaultMac,
		remoteProtocolVersion: 0,
		packetBufferSize:      2000,
	}
}

func (s *Socket) SetPacketHandler(packetHandler PacketHandler) {
	s.packetHandler = packetHandler
}

func (s *Socket) GetConnectionID() string {
	return s.connId
}

func (s *Socket) Wait() {
	s.wg.Wait()
}

func (s *Socket) closeDone() {
	s.wg.Done()
	s.Close()
}

func (s *Socket) Close() {
	s.adapter.Close()
	if s.iface != nil && s.ifaceManaged {
		s.iface.Close()
	}

	if s.closechanopen {
		s.closechanopen = false
		close(s.closechan)
	}

	if s.packetHandler != nil {
		s.packetHandler.UnregisterSocket(s)
	}
}

func (s *Socket) Serve() {
	s.registerDefaultCommandHandlers()

	if s.packetHandler != nil {
		s.packetHandler.RegisterSocket(s)
	}

	s.registerDataHandler()

	s.registerControlMessageHandler()

	s.installPingPongHandlers(*pingIntervalPtr, *pingTimeoutPtr)

	s.wg.Add(1)
	go func() {
		defer s.closeDone()
		err, unexpected := s.adapter.Serve()
		if unexpected {
			log.Printf("[%s] Client ERROR: %v", s.connId, err)
		}
	}()

	s.adapter.WaitReady()

	s.tryServeIfaceRead()

	go s.sendDefaultWelcome()
}
