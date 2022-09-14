package sockets

import (
	"log"
	"net"
	"sync"
	"time"

	"github.com/Doridian/wsvpn/shared"
	"github.com/Doridian/wsvpn/shared/commands"
	"github.com/Doridian/wsvpn/shared/features"
	"github.com/Doridian/wsvpn/shared/iface"
	"github.com/Doridian/wsvpn/shared/sockets/adapters"
)

const UndeterminedProtocolVersion = 0
const fragmentationMinProtocol = 10
const fragmentationNegotiatedMinProtocol = 11
const featureFieldMinProtocol = 12

type EventPusher = func(evt string)

type Socket struct {
	AssignedIP net.IP

	lastFragmentID        uint32
	lastFragmentCleanup   time.Time
	defragBuffer          map[uint32]*fragmentsInfo
	defragLock            *sync.Mutex
	fragmentCleanupTicker *time.Ticker
	fragmentationEnabled  bool

	compressionEnabled bool

	remoteProtocolVersion int

	adapter          adapters.SocketAdapter
	iface            *iface.WaterInterfaceWrapper
	ifaceManaged     bool
	wg               *sync.WaitGroup
	readyWait        *sync.Cond
	handlers         map[string]CommandHandler
	closechan        chan bool
	closechanopen    bool
	mac              net.HardwareAddr
	packetBufferSize int
	packetHandler    PacketHandler
	log              *log.Logger
	pingInterval     time.Duration
	pingTimeout      time.Duration
	isReady          bool
	isClosing        bool
	closeLock        *sync.Mutex

	localFeatures  map[features.Feature]bool
	remoteFeatures map[features.Feature]bool
	usedFeatures   map[features.Feature]bool

	eventPusher EventPusher
	upEventSent bool
}

func MakeSocket(logger *log.Logger, adapter adapters.SocketAdapter, iface *iface.WaterInterfaceWrapper, ifaceManaged bool, eventPusher EventPusher) *Socket {
	return &Socket{
		AssignedIP: net.IPv6unspecified,
		mac:        shared.DefaultMAC,

		adapter:               adapter,
		iface:                 iface,
		ifaceManaged:          ifaceManaged,
		wg:                    &sync.WaitGroup{},
		readyWait:             shared.MakeSimpleCond(),
		handlers:              make(map[commands.CommandName]CommandHandler),
		closechan:             make(chan bool),
		closechanopen:         true,
		remoteProtocolVersion: UndeterminedProtocolVersion,
		packetBufferSize:      2000,
		log:                   logger,
		isReady:               false,
		isClosing:             false,
		closeLock:             &sync.Mutex{},

		lastFragmentID:        0,
		defragBuffer:          make(map[uint32]*fragmentsInfo),
		defragLock:            &sync.Mutex{},
		lastFragmentCleanup:   time.Now(),
		fragmentCleanupTicker: time.NewTicker(fragmentExpiryTime),
		fragmentationEnabled:  false,

		compressionEnabled: false,

		eventPusher: eventPusher,

		localFeatures:  make(map[features.Feature]bool, 0),
		remoteFeatures: make(map[features.Feature]bool, 0),
		usedFeatures:   make(map[features.Feature]bool),
	}
}

func (s *Socket) ConfigurePing(pingInterval time.Duration, pingTimeout time.Duration) {
	s.pingInterval = pingInterval
	s.pingTimeout = pingTimeout
}

func (s *Socket) SetLocalFeature(feature features.Feature, enabled bool) {
	if !enabled {
		delete(s.localFeatures, feature)
		return
	}
	s.localFeatures[feature] = true
}

func (s *Socket) IsLocalFeature(feature features.Feature) bool {
	return s.localFeatures[feature]
}

func (s *Socket) IsFeatureEnabled(feature features.Feature) bool {
	return s.usedFeatures[feature]
}

func (s *Socket) SetPacketHandler(packetHandler PacketHandler) {
	s.packetHandler = packetHandler
}

func (s *Socket) HandleInitPacketFragmentation(enabled bool) {
	if s.remoteProtocolVersion >= featureFieldMinProtocol {
		return
	}

	s.SetLocalFeature(features.Fragmentation, true)
	if enabled {
		s.remoteFeatures[features.Fragmentation] = true
	} else {
		delete(s.remoteFeatures, features.Fragmentation)
	}

	s.featureCheck()
}

func (s *Socket) featureCheck() {
	if s.remoteProtocolVersion == UndeterminedProtocolVersion {
		return
	}

	s.usedFeatures = make(map[features.Feature]bool)
	for feat, en := range s.localFeatures {
		if !en {
			continue
		}
		if s.remoteFeatures[feat] {
			s.usedFeatures[feat] = true
		}
	}

	if s.remoteProtocolVersion >= fragmentationMinProtocol && s.remoteProtocolVersion < fragmentationNegotiatedMinProtocol {
		s.fragmentationEnabled = true
	} else if s.remoteProtocolVersion >= fragmentationNegotiatedMinProtocol && s.remoteProtocolVersion < featureFieldMinProtocol {
		s.fragmentationEnabled = s.localFeatures[features.Fragmentation]
	} else {
		s.fragmentationEnabled = s.IsFeatureEnabled(features.Fragmentation)
	}

	s.compressionEnabled = s.IsFeatureEnabled(features.Compression)

	s.log.Printf("Setting fragmentation: %s", shared.BoolToEnabled(s.fragmentationEnabled))
	// s.log.Printf("Setting compression: %s", shared.BoolToEnabled(s.compressionEnabled))

	if s.adapter != nil {
		s.adapter.RefreshFeatures()
	}
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
		_ = s.SendMessage("error", err.Error())
	}
	s.Close()
}

func (s *Socket) setReady() {
	s.isReady = true
	s.readyWait.Broadcast()
}

func (s *Socket) Close() {
	s.closeLock.Lock()
	defer s.closeLock.Unlock()

	_ = s.adapter.Close()
	if s.iface != nil && s.ifaceManaged {
		_ = s.iface.Close()
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

	if s.eventPusher != nil && s.upEventSent {
		s.upEventSent = false
		s.eventPusher(shared.EventDown)
	}
}

func (s *Socket) Serve() {
	s.registerDefaultCommandHandlers()

	if s.packetHandler != nil {
		s.packetHandler.RegisterSocket(s)
	}

	s.adapter.SetFeaturesContainer(s)
	s.adapter.SetDataMessageHandler(s.dataMessageHandler)
	s.registerControlMessageHandler()
	s.installPingPongHandlers()

	s.wg.Add(1)
	go func() {
		defer s.closeDone()
		unexpected, err := s.adapter.Serve()
		if unexpected {
			s.log.Printf("Adapter ERROR: %v", err)
		}
	}()

	s.adapter.WaitReady()

	s.tryServeIfaceRead()
	go s.cleanupFragmentsLoop()
	go s.sendDefaultWelcome()

	if s.eventPusher != nil {
		s.closeLock.Lock()
		if s.closechanopen {
			s.upEventSent = true
			s.eventPusher(shared.EventUp)
		}
		s.closeLock.Unlock()
	}
}
