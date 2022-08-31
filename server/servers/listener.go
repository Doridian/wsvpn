package servers

import (
	"crypto/tls"
	"errors"
	"net/http"

	"github.com/Doridian/wsvpn/server/upgraders"
	"github.com/Doridian/wsvpn/shared"
)

func (s *Server) listenUpgraders() {
	for _, upgraderLoop := range s.upgraders {
		s.serveWaitGroup.Add(1)
		go func(upgrader upgraders.SocketUpgrader) {
			defer s.serveWaitGroup.Done()
			err := upgrader.ListenAndServe()
			s.setServeError(err)
		}(upgraderLoop)
	}
}

func (s *Server) addUpgrader(upgrader upgraders.SocketUpgrader) {
	s.upgraders = append(s.upgraders, upgrader)
	s.addCloser(upgrader)
}

func (s *Server) listenPlaintext(httpHandlerFunc http.HandlerFunc) {
	if s.HTTP3Enabled {
		s.setServeError(errors.New("HTTP/3 requires TLS"))
		return
	}

	s.addUpgrader(upgraders.NewWebSocketUpgrader())

	s.listenUpgraders()

	server := &http.Server{
		Addr:    s.ListenAddr,
		Handler: httpHandlerFunc,
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
			webtransportUpgrader.SetQuicHeaders(w.Header())
			s.serveSocket(w, r)
		})
	}

	server := &http.Server{
		Addr:      s.ListenAddr,
		TLSConfig: s.TLSConfig,
		Handler:   httpHandlerFunc,
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

	s.log.Printf("VPN server online at %s (HTTP/3 %s, TLS %s, mTLS %s), Mode %s, Subnet %s (%d max clients), MTU %d",
		s.ListenAddr, shared.BoolToEnabled(s.HTTP3Enabled), shared.BoolToEnabled(s.TLSConfig != nil),
		shared.BoolToEnabled(s.TLSConfig != nil && s.TLSConfig.ClientAuth == tls.RequireAndVerifyClientCert), s.Mode.ToString(), s.VPNNet.GetRaw(), s.VPNNet.GetClientSlots(), s.mtu)

	httpHandlerFunc := http.HandlerFunc(s.serveSocket)

	if s.TLSConfig == nil {
		s.listenPlaintext(httpHandlerFunc)
	} else {
		s.listenEncrypted(httpHandlerFunc)
	}
}
