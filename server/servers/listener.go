package servers

import (
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/Doridian/wsvpn/server/upgraders"
	"github.com/Doridian/wsvpn/shared"
)

const ReadHeaderTimeout = time.Duration(10) * time.Second

func (s *Server) listenUpgraders() error {
	for _, upgrader := range s.upgraders {
		upgrader.SetHeaders(s.headers)
		err := upgrader.Listen()
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) serveUpgraders() {
	for _, upgrader := range s.upgraders {
		s.serveWaitGroup.Add(1)
		go func(upgrader upgraders.SocketUpgrader) {
			defer s.serveWaitGroup.Done()
			err := upgrader.Serve()
			s.setServeError(err)
		}(upgrader)
	}
}

func (s *Server) addUpgrader(upgrader upgraders.SocketUpgrader) {
	s.upgraders = append(s.upgraders, upgrader)
	s.addCloser(upgrader)
}

func (s *Server) setUpgraderHeaders() {
	for _, upgrader := range s.upgraders {
		upgrader.SetHeaders(s.headers)
	}
}

func (s *Server) listenPlaintext(httpHandlerFunc http.HandlerFunc) (func(), error) {
	if s.HTTP3Enabled {
		return nil, errors.New("HTTP/3 requires TLS")
	}

	s.addUpgrader(upgraders.NewWebSocketUpgrader())

	server := &http.Server{
		Addr:              s.ListenAddr,
		Handler:           httpHandlerFunc,
		ReadHeaderTimeout: ReadHeaderTimeout,
	}
	s.addCloser(server)

	ln, err := net.Listen("tcp", s.ListenAddr)
	if err != nil {
		return nil, err
	}

	return func() {
		s.serveWaitGroup.Add(1)
		go func() {
			err := server.Serve(ln)
			s.setServeError(err)
		}()
	}, nil
}

func (s *Server) listenEncrypted(httpHandlerFunc http.HandlerFunc) (func(), error) {
	if s.HTTP3Enabled {
		webtransportUpgrader := upgraders.NewWebTransportUpgrader(&upgraders.QuicServerConfig{
			Addr:      s.ListenAddr,
			TLSConfig: s.TLSConfig,
			Handler:   httpHandlerFunc,
		})
		s.addUpgrader(webtransportUpgrader) // This calls addCloser for us

		httpHandlerFunc = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_ = webtransportUpgrader.SetQUICHeaders(w.Header())
			s.serveSocket(w, r)
		})
	}

	server := &http.Server{
		Addr:              s.ListenAddr,
		TLSConfig:         s.TLSConfig,
		Handler:           httpHandlerFunc,
		ReadHeaderTimeout: ReadHeaderTimeout,
	}
	s.addCloser(server)

	s.addUpgrader(upgraders.NewWebSocketUpgrader())

	ln, err := net.Listen("tcp", s.ListenAddr)
	if err != nil {
		return nil, err
	}

	return func() {
		s.serveWaitGroup.Add(1)
		go func() {
			err := server.ServeTLS(ln, "", "")
			s.setServeError(err)
		}()
	}, nil
}

func (s *Server) listen() error {
	s.upgraders = make([]upgraders.SocketUpgrader, 0)

	tlsConfigTemp := s.TLSConfig
	if tlsConfigTemp != nil && tlsConfigTemp.GetConfigForClient != nil {
		tlsConfigTemp, _ = tlsConfigTemp.GetConfigForClient(nil)
	}

	httpHandlerFunc := http.HandlerFunc(s.serveSocket)

	var serveHTTP func()
	var err error
	if tlsConfigTemp == nil {
		serveHTTP, err = s.listenPlaintext(httpHandlerFunc)
	} else {
		serveHTTP, err = s.listenEncrypted(httpHandlerFunc)
	}
	if err != nil {
		return err
	}

	err = s.listenUpgraders()
	if err != nil {
		return err
	}

	// Only now that every listener is bound are we actually reachable
	s.log.Printf("VPN server online at %s (HTTP/3 %s, TLS %s, mTLS %s), Mode %s, Subnet %s (%d max clients), MTU %d",
		s.ListenAddr, shared.BoolToEnabled(s.HTTP3Enabled), shared.BoolToEnabled(tlsConfigTemp != nil),
		shared.BoolToEnabled(tlsConfigTemp != nil && tlsConfigTemp.ClientAuth == tls.RequireAndVerifyClientCert), s.Mode.ToString(), s.VPNNet.GetRaw(), s.VPNNet.GetClientSlots(), s.mtu)

	serveHTTP()
	s.serveUpgraders()
	return nil
}
