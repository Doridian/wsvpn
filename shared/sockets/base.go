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

const UndeterminedProtocolVersion = 0

type Socket struct {
	lastFragmentId        uint32
	lastFragmentCleanup   time.Time
	defragBuffer          map[uint32]*fragmentsInfo
	defragLock            *sync.Mutex
	fragmentCleanupTicker *time.Ticker
	fragmentationEnabled  bool

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
	isClosing             bool
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
		remoteProtocolVersion: UndeterminedProtocolVersion,
		packetBufferSize:      2000,
		log:                   logger,
		isReady:               false,
		isClosing:             false,

		lastFragmentId:        0,
		defragBuffer:          make(map[uint32]*fragmentsInfo),
		defragLock:            &sync.Mutex{},
		lastFragmentCleanup:   time.Now(),
		fragmentCleanupTicker: time.NewTicker(fragmentExpiryTime),
	}
}

func (s *Socket) ConfigurePing(pingInterval time.Duration, pingTimeout time.Duration) {
	s.pingInterval = pingInterval
	s.pingTimeout = pingTimeout
}

func (s *Socket) SetPacketHandler(packetHandler PacketHandler) {
	s.packetHandler = packetHandler
}

func (s *Socket) SetEnableFragmentationAlways(enabled bool) {
	s.fragmentationEnabled = enabled
}

func (s *Socket) SetEnableFragmentationIfSupported(enabled bool) {
	if s.remoteProtocolVersion < fragmentationNegotiatedMinProtocol {
		return
	}
	s.fragmentationEnabled = enabled
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
	if !s.isClosing {
		s.isClosing = true
		s.log.Printf("Closing socket: %v", err)
		s.SendMessage("error", err.Error())
	}
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

	s.fragmentCleanupTicker.Stop()
}

func (s *Socket) Serve() {
	s.registerDefaultCommandHandlers()

	if s.packetHandler != nil {
		s.packetHandler.RegisterSocket(s)
	}

	s.adapter.SetDataMessageHandler(s.dataMessageHandler)

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

	go s.cleanupFragmentsLoop()

	s.tryServeIfaceRead()

	go s.sendDefaultWelcome()
}
