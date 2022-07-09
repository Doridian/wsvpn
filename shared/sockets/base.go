package sockets

import (
	"log"
	"sync"
	"time"

	"github.com/Doridian/wsvpn/shared"
	"github.com/Doridian/wsvpn/shared/commands"
	"github.com/Doridian/wsvpn/shared/sockets/adapters"
	"github.com/songgao/water"
)

type Socket struct {
	adapter               adapters.SocketAdapter
	iface                 *water.Interface
	ifaceManaged          bool
	wg                    *sync.WaitGroup
	readyWait             *sync.Cond
	handlers              map[string]CommandHandler
	closechan             chan bool
	closechanopen         bool
	mac                   shared.MacAddr
	remoteProtocolVersion int
	packetBufferSize      int
	packetHandler         PacketHandler
	log                   *log.Logger
	pingInterval          time.Duration
	pingTimeout           time.Duration
	isReady               bool
}

func MakeSocket(logger *log.Logger, adapter adapters.SocketAdapter, iface *water.Interface, ifaceManaged bool) *Socket {
	return &Socket{
		adapter:               adapter,
		iface:                 iface,
		ifaceManaged:          ifaceManaged,
		wg:                    &sync.WaitGroup{},
		readyWait:             shared.MakeSimpleCond(),
		handlers:              make(map[commands.CommandName]CommandHandler),
		closechan:             make(chan bool),
		closechanopen:         true,
		mac:                   shared.DefaultMac,
		remoteProtocolVersion: 0,
		packetBufferSize:      2000,
		log:                   logger,
		isReady:               false,
	}
}

func (s *Socket) ConfigurePing(pingInterval time.Duration, pingTimeout time.Duration) {
	s.pingInterval = pingInterval
	s.pingTimeout = pingTimeout
}

func (s *Socket) SetPacketHandler(packetHandler PacketHandler) {
	s.packetHandler = packetHandler
}

func (s *Socket) Wait() {
	s.wg.Wait()
}

func (s *Socket) WaitReady() {
	for !s.isReady {
		s.readyWait.L.Lock()
		s.readyWait.Wait()
		s.readyWait.L.Unlock()
	}
}

func (s *Socket) closeDone() {
	s.wg.Done()
	s.Close()
}

func (s *Socket) CloseError(err error) {
	s.log.Printf("Closing socket: %v", err)
	s.SendMessage("error", err.Error())
	s.Close()
}

func (s *Socket) setReady() {
	s.isReady = true
	s.readyWait.Broadcast()
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

	s.setReady()
}

func (s *Socket) Serve() {
	s.registerDefaultCommandHandlers()

	if s.packetHandler != nil {
		s.packetHandler.RegisterSocket(s)
	}

	s.registerDataHandler()

	s.registerControlMessageHandler()

	s.installPingPongHandlers()

	s.wg.Add(1)
	go func() {
		defer s.closeDone()
		err, unexpected := s.adapter.Serve()
		if unexpected {
			s.log.Printf("Adapter ERROR: %v", err)
		}
	}()

	s.adapter.WaitReady()

	s.tryServeIfaceRead()

	go s.sendDefaultWelcome()
}
