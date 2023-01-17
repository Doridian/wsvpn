package servers

import (
	"crypto/tls"
	"errors"
	"io"
	"log"
	"sync"

	"github.com/Doridian/wsvpn/server/authenticators"
	"github.com/Doridian/wsvpn/server/upgraders"
	"github.com/Doridian/wsvpn/shared"
	"github.com/Doridian/wsvpn/shared/commands"
	"github.com/Doridian/wsvpn/shared/features"
	"github.com/Doridian/wsvpn/shared/iface"
	"github.com/Doridian/wsvpn/shared/sockets"
)

var errNone = errors.New("none")
var ErrNoServeWaitsLeft = errors.New("no serve waits left")

type MaxConnectionsPerUserEnum = int

const (
	MaxConnectionsPerUserKillOldest MaxConnectionsPerUserEnum = 0
	MaxConnectionsPerUserPreventNew MaxConnectionsPerUserEnum = 1
)

type Server struct {
	shared.EventConfigHolder

	PacketHandler             sockets.PacketHandler
	VPNNet                    *shared.VPNNet
	DoLocalIPConfig           bool
	DoRemoteIPConfig          bool
	TLSConfig                 *tls.Config
	ListenAddr                string
	HTTP3Enabled              bool
	Authenticator             authenticators.Authenticator
	Mode                      shared.VPNMode
	SocketConfigurator        sockets.SocketConfigurator
	InterfaceConfig           *iface.InterfaceConfig
	MaxConnectionsPerUser     int
	MaxConnectionsPerUserMode MaxConnectionsPerUserEnum
	WebsiteDirectory          string
	APIEnabled                bool
	APIUsers                  map[string]bool
	PreauthorizeSecret        []byte

	upgraders          []upgraders.SocketUpgrader
	slotMutex          *sync.Mutex
	ifaceCreationMutex *sync.Mutex
	usedSlots          map[uint64]bool
	packetBufferSize   int
	mtu                int
	mainIface          *iface.WaterInterfaceWrapper
	log                *log.Logger
	serverID           string

	closers              []io.Closer
	sockets              map[string]*sockets.Socket
	authenticatedSockets map[string][]*sockets.Socket
	closerLock           *sync.Mutex
	socketsLock          *sync.Mutex

	serveErrorChannel chan interface{}
	serveError        error
	serveWaitGroup    *sync.WaitGroup

	localFeatures map[features.Feature]bool
}

func NewServer() *Server {
	return &Server{
		slotMutex:            &sync.Mutex{},
		ifaceCreationMutex:   &sync.Mutex{},
		usedSlots:            make(map[uint64]bool),
		log:                  shared.MakeLogger("SERVER", ""),
		serveErrorChannel:    make(chan interface{}),
		serveWaitGroup:       &sync.WaitGroup{},
		closers:              make([]io.Closer, 0),
		sockets:              make(map[string]*sockets.Socket),
		authenticatedSockets: make(map[string][]*sockets.Socket),
		closerLock:           &sync.Mutex{},
		socketsLock:          &sync.Mutex{},
		localFeatures:        make(map[features.Feature]bool),
	}
}

func (s *Server) SetServerID(serverID string) {
	s.serverID = serverID
	shared.UpdateLogger(s.log, "SERVER", s.serverID)
}

func (s *Server) addCloser(closer io.Closer) {
	s.closerLock.Lock()
	defer s.closerLock.Unlock()
	s.closers = append(s.closers, closer)
}

func (s *Server) setServeError(err error) {
	if err == nil {
		return
	}

	if s.serveError != nil {
		return
	}
	s.serveError = err
	close(s.serveErrorChannel)
}

func (s *Server) Serve() error {
	err := iface.VerifyPlatformFlags(s.InterfaceConfig, s.Mode)
	if err != nil {
		return err
	}

	if !s.InterfaceConfig.OneInterfacePerConnection {
		err = s.createMainIface()
		if err != nil {
			return err
		}

		s.serveWaitGroup.Add(1)
		s.addCloser(s.mainIface)
		go s.serveMainIface()
	}

	s.listen()

	go func() {
		s.serveWaitGroup.Wait()
		s.setServeError(ErrNoServeWaitsLeft)
	}()

	<-s.serveErrorChannel

	s.closeAll()

	if s.serveError == errNone {
		return nil
	}
	return s.serveError
}

func (s *Server) closeAll() {
	s.closerLock.Lock()
	defer s.closerLock.Unlock()

	for _, closer := range s.closers {
		_ = closer.Close()
	}
	s.closers = make([]io.Closer, 0)
}

func (s *Server) Close() {
	s.closeAll()
	s.setServeError(errNone)
}

func (s *Server) SetMTU(mtu int) error {
	if mtu < 500 || mtu > 65535 {
		return errors.New("MTU out of range (500 - 65535)")
	}
	if s.mtu == mtu {
		return nil
	}

	if s.mainIface != nil {
		err := s.mainIface.SetMTU(mtu)
		if err != nil {
			return err
		}
	}

	s.packetBufferSize = shared.GetPacketBufferSizeByMTU(mtu)

	s.socketsLock.Lock()
	defer s.socketsLock.Unlock()

	for _, sock := range s.sockets {
		sock.SetMTU(mtu)
		iface := sock.GetInterfaceIfManaged()
		if iface != nil {
			err := iface.SetMTU(mtu)
			if err != nil {
				return err
			}
		}
		_ = sock.MakeAndSendCommand(&commands.SetMTUParameters{
			MTU: mtu,
		})
	}

	s.mtu = mtu

	return nil
}

func (s *Server) SetLocalFeature(feature features.Feature, enabled bool) {
	if !enabled {
		delete(s.localFeatures, feature)
		return
	}
	s.localFeatures[feature] = true
}
