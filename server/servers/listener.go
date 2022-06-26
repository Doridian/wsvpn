package servers

import (
	"crypto/tls"
	"errors"
	"net/http"
	"sync"

	"github.com/Doridian/wsvpn/shared"
	"github.com/Doridian/wsvpn/shared/sockets/upgraders"
	"github.com/lucas-clemente/quic-go/http3"
)

func (s *Server) listenPlaintext() error {
	httpHandlerFunc := http.HandlerFunc(s.serveSocket)

	if s.HTTP3Enabled {
		return errors.New("HTTP/3 requires TLS")
	}
	server := http.Server{
		Addr:    s.ListenAddr,
		Handler: httpHandlerFunc,
	}
	return server.ListenAndServe()
}

func (s *Server) listenEncrypted() error {
	httpHandlerFunc := http.HandlerFunc(s.serveSocket)

	listenerWaitGroup := &sync.WaitGroup{}

	if s.HTTP3Enabled {
		listenerWaitGroup.Add(1)

		quicServer := &http3.Server{
			Addr:      s.ListenAddr,
			TLSConfig: s.TLSConfig,
			Handler:   httpHandlerFunc,
		}

		s.upgraders = append(s.upgraders, upgraders.NewWebTransportUpgrader(quicServer))

		httpHandlerFunc = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			quicServer.SetQuicHeaders(w.Header())
			s.serveSocket(w, r)
		})

		// This should be at the end of this func, but WebTransport calls it for us, and there is no way to avoid it
		// err := quicServer.ListenAndServe()
		// if err != nil {
		//	return err
		// }
	}

	server := http.Server{
		Addr:      s.ListenAddr,
		TLSConfig: s.TLSConfig,
		Handler:   httpHandlerFunc,
	}

	s.upgraders = append(s.upgraders, upgraders.NewWebSocketUpgrader())

	for _, upgraderLoop := range s.upgraders {
		listenerWaitGroup.Add(1)
		go func(upgrader upgraders.SocketUpgrader) {
			defer listenerWaitGroup.Done()
			err := upgrader.ListenAndServe()
			if err != nil {
				panic(err) // TODO: This should not panic
			}
		}(upgraderLoop)
	}

	err := server.ListenAndServeTLS("", "")
	if err != nil {
		return err
	}

	listenerWaitGroup.Wait()
	return nil
}

func (s *Server) Listen() error {
	s.upgraders = make([]upgraders.SocketUpgrader, 0)

	s.log.Printf("VPN server online at %s (HTTP/3 %s, TLS %s, mTLS %s), Mode %s, Subnet %s (%d max clients), MTU %d",
		s.ListenAddr, shared.BoolToEnabled(s.HTTP3Enabled), shared.BoolToEnabled(s.TLSConfig != nil),
		shared.BoolToEnabled(s.TLSConfig != nil && s.TLSConfig.ClientAuth == tls.RequireAndVerifyClientCert), s.Mode.ToString(), s.VPNNet.GetRaw(), s.VPNNet.GetClientSlots(), s.mtu)

	if s.TLSConfig == nil {
		return s.listenPlaintext()
	} else {
		return s.listenEncrypted()
	}
}
