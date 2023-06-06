package servers

import (
	"crypto/tls"
	"errors"
	"net/http"
	"time"

	"github.com/Doridian/wsvpn/server/upgraders"
	"github.com/Doridian/wsvpn/shared"
)

const ReadHeaderTimeout = time.Duration(10) * time.Second

func (s *Server) listenUpgraders() {
	for _, upgrader := range s.upgraders {
		s.serveWaitGroup.Add(1)
		upgrader.SetHeaders(s.headers)
		go func(upgrader upgraders.SocketUpgrader) {
			defer s.serveWaitGroup.Done()
			err := upgrader.ListenAndServe()
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

func (s *Server) listenPlaintext(httpHandlerFunc http.HandlerFunc) {
	if s.HTTP3Enabled {
		s.setServeError(errors.New("HTTP/3 requires TLS"))
		return
	}

	s.addUpgrader(upgraders.NewWebSocketUpgrader())

	s.listenUpgraders()

	server := &http.Server{
		Addr:              s.ListenAddr,
		Handler:           httpHandlerFunc,
		ReadHeaderTimeout: ReadHeaderTimeout,
	}
	s.addCloser(server)

	s.serveWaitGroup.Add(1)
	go func() {
		err := server.ListenAndServe()
		s.setServeError(err)
	}()
}

func (s *Server) listenEncrypted(httpHandlerFunc http.HandlerFunc) {
	if s.HTTP3Enabled {
		webtransportUpgrader := upgraders.NewWebTransportUpgrader(&upgraders.QuicServerConfig{
			Addr:      s.ListenAddr,
			TLSConfig: s.TLSConfig,
			Handler:   httpHandlerFunc,
		})
		s.addUpgrader(webtransportUpgrader) // This calls addCloser for us

		httpHandlerFunc = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_ = webtransportUpgrader.SetQuicHeaders(w.Header())
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
	s.listenUpgraders()

	s.serveWaitGroup.Add(1)
	go func() {
		err := server.ListenAndServeTLS("", "")
		s.setServeError(err)
	}()
}

func (s *Server) listen() {
	s.upgraders = make([]upgraders.SocketUpgrader, 0)

	tlsConfigTemp, _ := s.TLSConfig.GetConfigForClient(nil)

	s.log.Printf("VPN server online at %s (HTTP/3 %s, TLS %s, mTLS %s), Mode %s, Subnet %s (%d max clients), MTU %d",
		s.ListenAddr, shared.BoolToEnabled(s.HTTP3Enabled), shared.BoolToEnabled(s.TLSConfig != nil),
		shared.BoolToEnabled(tlsConfigTemp != nil && tlsConfigTemp.ClientAuth == tls.RequireAndVerifyClientCert), s.Mode.ToString(), s.VPNNet.GetRaw(), s.VPNNet.GetClientSlots(), s.mtu)

	httpHandlerFunc := http.HandlerFunc(s.serveSocket)

	if s.TLSConfig == nil {
		s.listenPlaintext(httpHandlerFunc)
	} else {
		s.listenEncrypted(httpHandlerFunc)
	}
}
