package servers

import (
	"crypto/tls"
	"log"
	"sync"

	"github.com/Doridian/wsvpn/server/authenticators"
	"github.com/Doridian/wsvpn/shared"
	"github.com/Doridian/wsvpn/shared/sockets/groups"
	"github.com/gorilla/websocket"
	"github.com/marten-seemann/webtransport-go"
	"github.com/songgao/water"
)

type Server struct {
	webSocketUpgrader  *websocket.Upgrader
	webTransportServer *webtransport.Server

	slotMutex          *sync.Mutex
	ifaceCreationMutex *sync.Mutex
	usedSlots          map[uint64]bool
	packetBufferSize   int
	mtu                int
	tapIface           *water.Interface
	log                *log.Logger
	serverId           string

	SocketGroup      *groups.SocketGroup
	VPNNet           *shared.VPNNet
	DoLocalIpConfig  bool
	DoRemoteIpConfig bool
	TLSConfig        *tls.Config
	ListenAddr       string
	HTTP3Enabled     bool
	Authenticator    authenticators.Authenticator
	Mode             shared.VPNMode
}

func NewServer() *Server {
	return &Server{
		slotMutex:          &sync.Mutex{},
		ifaceCreationMutex: &sync.Mutex{},
		usedSlots:          make(map[uint64]bool),
		log:                shared.MakeLogger("SERVER", "UNSET"),
	}
}

func (s *Server) SetServerID(serverId string) {
	s.serverId = serverId
	shared.UpdateLogger(s.log, "SERVER", s.serverId)
}

func (s *Server) Serve() error {
	err := s.verifyPlatformFlags()
	if err != nil {
		return err
	}

	if s.Mode == shared.VPN_MODE_TAP {
		err := s.createTAP()
		if err != nil {
			return err
		}
		go s.serveTAP()
	}

	return s.Listen()
}

func (s *Server) SetMTU(mtu int) {
	s.mtu = mtu
	s.packetBufferSize = shared.GetPacketBufferSizeByMTU(mtu)
}
