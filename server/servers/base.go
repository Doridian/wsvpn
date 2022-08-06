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
	"github.com/Doridian/wsvpn/shared/sockets"
	"github.com/songgao/water"
)

var errNone = errors.New("none")
var ErrNoServeWaitsLeft = errors.New("no serve waits left")

type Server struct {
	shared.EventConfigHolder

	PacketHandler       sockets.PacketHandler
	VPNNet              *shared.VPNNet
	DoLocalIpConfig     bool
	DoRemoteIpConfig    bool
	TLSConfig           *tls.Config
	ListenAddr          string
	HTTP3Enabled        bool
	Authenticator       authenticators.Authenticator
	Mode                shared.VPNMode
	SocketConfigurator  sockets.SocketConfigurator
	InterfaceConfig     *shared.InterfaceConfig
	EnableFragmentation bool

	upgraders          []upgraders.SocketUpgrader
	slotMutex          *sync.Mutex
	ifaceCreationMutex *sync.Mutex
	usedSlots          map[uint64]bool
	packetBufferSize   int
	mtu                int
	mainIface          *water.Interface
	log                *log.Logger
	serverId           string

	closers    []io.Closer
	closerLock *sync.Mutex

	serveErrorChannel chan interface{}
	serveError        error
	serveWaitGroup    *sync.WaitGroup
}

func NewServer() *Server {
	return &Server{
		slotMutex:          &sync.Mutex{},
		ifaceCreationMutex: &sync.Mutex{},
		usedSlots:          make(map[uint64]bool),
		log:                shared.MakeLogger("SERVER", ""),
		serveErrorChannel:  make(chan interface{}),
		serveWaitGroup:     &sync.WaitGroup{},
		closers:            make([]io.Closer, 0),
		closerLock:         &sync.Mutex{},
	}
}

func (s *Server) SetServerID(serverId string) {
	s.serverId = serverId
	shared.UpdateLogger(s.log, "SERVER", s.serverId)
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

	if s.serveError == nil {
		s.serveError = err
	}
	close(s.serveErrorChannel)
}

func (s *Server) Serve() error {
	err := s.verifyPlatformFlags()
	if err != nil {
		return err
	}

	err = s.createMainIface()
	if err != nil {
		return err
	}

	s.serveWaitGroup.Add(1)
	s.addCloser(s.mainIface)
	go s.serveMainIface()

	s.listen()

	go func() {
		s.serveWaitGroup.Wait()
		s.setServeError(ErrNoServeWaitsLeft)
	}()

	<-s.serveErrorChannel

	s.closerLock.Lock()
	for _, closer := range s.closers {
		closer.Close()
	}
	s.closers = make([]io.Closer, 0)
	s.closerLock.Unlock()

	if s.serveError == errNone {
		return nil
	}
	return s.serveError
}

func (s *Server) Close() {
	s.setServeError(errNone)
}

func (s *Server) SetMTU(mtu int) {
	s.mtu = mtu
	s.packetBufferSize = shared.GetPacketBufferSizeByMTU(mtu)
}
