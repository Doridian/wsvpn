package servers

import (
	"crypto/tls"
	"errors"
	"net/http"
	"sync"

	"github.com/Doridian/wsvpn/shared"
	"github.com/gorilla/websocket"
	"github.com/lucas-clemente/quic-go/http3"
	"github.com/marten-seemann/webtransport-go"
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

	http3Wait := &sync.WaitGroup{}

	if s.HTTP3Enabled {
		http3Wait.Add(1)

		quicServer := http3.Server{
			Addr:      s.ListenAddr,
			TLSConfig: s.TLSConfig,
			Handler:   httpHandlerFunc,
		}

		s.webTransportServer = &webtransport.Server{
			H3:          quicServer,
			CheckOrigin: func(r *http.Request) bool { return true },
		}

		go func() {
			defer http3Wait.Done()
			err := s.webTransportServer.ListenAndServe()
			if err != nil {
				panic(err) // TODO: This should not panic
			}
		}()

		httpHandlerFunc = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			quicServer.SetQuicHeaders(w.Header())
			s.serveSocket(w, r)
		})
	}

	server := http.Server{
		Addr:      s.ListenAddr,
		TLSConfig: s.TLSConfig,
		Handler:   httpHandlerFunc,
	}

	err := server.ListenAndServeTLS("", "")
	if err != nil {
		return err
	}

	http3Wait.Wait()
	return nil
}

func (s *Server) Listen() error {
	s.log.Printf("VPN server online at %s (HTTP/3 %s, TLS %s, mTLS %s), mode %s, serving subnet %s (%d max clients) with MTU %d",
		s.ListenAddr, shared.BoolToEnabled(s.HTTP3Enabled), shared.BoolToEnabled(s.TLSConfig != nil),
		shared.BoolToEnabled(s.TLSConfig != nil && s.TLSConfig.ClientAuth == tls.RequireAndVerifyClientCert), s.Mode.ToString(), s.VPNNet.GetRaw(), s.VPNNet.GetClientSlots(), s.mtu)

	s.webSocketUpgrader = &websocket.Upgrader{
		ReadBufferSize:  2048,
		WriteBufferSize: 2048,
		CheckOrigin:     func(r *http.Request) bool { return true },
	}

	if s.TLSConfig == nil {
		return s.listenPlaintext()
	} else {
		return s.listenEncrypted()
	}
}
