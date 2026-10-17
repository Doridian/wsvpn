package sockets

import (
	"log"
	"net"
	"sync"
	"sync/atomic"
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
	fragmentationEnabled  atomic.Bool

	compressionEnabled bool

	remoteProtocolVersion atomic.Int64

	adapter          adapters.SocketAdapter
	iface            *iface.WaterInterfaceWrapper
	ifaceManaged     bool
	wg               *sync.WaitGroup
	readyWait        *sync.Cond
	handlers         map[string]CommandHandler
	closeChan        chan bool
	closeChanOpen    bool
	mac              net.HardwareAddr
	packetBufferSize int
	packetHandler    PacketHandler
	log              *log.Logger
	pingInterval     time.Duration
	pingTimeout      time.Duration
	isReady          atomic.Bool
	isClosing        atomic.Bool
	closeLock        *sync.Mutex

	localFeatures  map[features.Feature]bool
	remoteFeatures map[features.Feature]bool
	usedFeatures   map[features.Feature]bool

	eventPusher EventPusher
	upEventSent bool

	Metadata map[string]interface{}
}

func MakeSocket(logger *log.Logger, adapter adapters.SocketAdapter, iface *iface.WaterInterfaceWrapper, ifaceManaged bool, eventPusher EventPusher) *Socket {
	return &Socket{
		AssignedIP: net.IPv6unspecified,
		mac:        shared.DefaultMAC,

		adapter:          adapter,
		iface:            iface,
		ifaceManaged:     ifaceManaged,
		wg:               &sync.WaitGroup{},
		readyWait:        shared.MakeSimpleCond(),
		handlers:         make(map[commands.CommandName]CommandHandler),
		closeChan:        make(chan bool),
		closeChanOpen:    true,
		packetBufferSize: 2000,
		log:              logger,
		closeLock:        &sync.Mutex{},

		lastFragmentID:        0,
		defragBuffer:          make(map[uint32]*fragmentsInfo),
		defragLock:            &sync.Mutex{},
		lastFragmentCleanup:   time.Now(),
		fragmentCleanupTicker: time.NewTicker(fragmentExpiryTime),

		compressionEnabled: false,

		eventPusher: eventPusher,

		localFeatures:  make(map[features.Feature]bool, 0),
		remoteFeatures: make(map[features.Feature]bool, 0),
		usedFeatures:   make(map[features.Feature]bool),

		Metadata: make(map[string]interface{}),
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

func (s *Socket) remoteVersion() int {
	return int(s.remoteProtocolVersion.Load())
}

func (s *Socket) HandleInitPacketFragmentation(enabled bool) {
	if s.remoteVersion() >= featureFieldMinProtocol {
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
	remoteProtocolVersion := s.remoteVersion()
	if remoteProtocolVersion == UndeterminedProtocolVersion {
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

	var fragmentationEnabled bool
	if remoteProtocolVersion >= fragmentationMinProtocol && remoteProtocolVersion < fragmentationNegotiatedMinProtocol {
		fragmentationEnabled = true
	} else if remoteProtocolVersion >= fragmentationNegotiatedMinProtocol && remoteProtocolVersion < featureFieldMinProtocol {
		fragmentationEnabled = s.localFeatures[features.Fragmentation]
	} else {
		fragmentationEnabled = s.IsFeatureEnabled(features.Fragmentation)
	}
	s.fragmentationEnabled.Store(fragmentationEnabled)

	s.compressionEnabled = s.IsFeatureEnabled(features.Compression)

	s.log.Printf("Setting fragmentation: %s", shared.BoolToEnabled(fragmentationEnabled))
	// s.log.Printf("Setting compression: %s", shared.BoolToEnabled(s.compressionEnabled))

	if s.adapter != nil {
		s.adapter.RefreshFeatures()
	}
}

func (s *Socket) Wait() {
	s.wg.Wait()
}

func (s *Socket) WaitReady() {
	s.readyWait.L.Lock()
	defer s.readyWait.L.Unlock()
	for !s.isReady.Load() {
		s.readyWait.Wait()
	}
}

func (s *Socket) closeDone() {
	s.wg.Done()
	s.Close()
}

func (s *Socket) CloseError(err error) {
	s.closeLock.Lock()
	defer s.closeLock.Unlock()

	wasClosing := s.isClosing.Load()
	s.close(false)

	if !wasClosing {
		s.log.Printf("Closing socket with error: %v", err)
		go func() {
			_ = s.SendMessage("error", err.Error())
			_ = s.adapter.Close()
		}()
	} else {
		_ = s.adapter.Close()
	}
}

func (s *Socket) setReady() {
	s.readyWait.L.Lock()
	s.isReady.Store(true)
	s.readyWait.Broadcast()
	s.readyWait.L.Unlock()
}

func (s *Socket) Close() {
	s.closeLock.Lock()
	defer s.closeLock.Unlock()
	if s.isClosing.Load() {
		return
	}
	s.close(true)
}

func (s *Socket) GetAdapter() adapters.SocketAdapter {
	return s.adapter
}

func (s *Socket) LocalAddr() net.Addr {
	return s.adapter.LocalAddr()
}

func (s *Socket) RemoteAddr() net.Addr {
	return s.adapter.RemoteAddr()
}

func (s *Socket) close(closeAdapter bool) {
	s.isClosing.Store(true)

	if closeAdapter {
		_ = s.adapter.Close()
	}

	if s.iface != nil && s.ifaceManaged {
		_ = s.iface.Close()
	}

	if s.closeChanOpen {
		s.closeChanOpen = false
		close(s.closeChan)
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
		if s.closeChanOpen {
			s.upEventSent = true
			s.eventPusher(shared.EventUp)
		}
		s.closeLock.Unlock()
	}
}
