package servers

import (
	"net/http"

	"github.com/Doridian/wsvpn/shared/sockets/adapters"
)

func (s *Server) serveWebSocket(w http.ResponseWriter, r *http.Request) (adapters.SocketAdapter, error) {
	conn, err := s.webSocketUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, err
	}
	return adapters.NewWebSocketAdapter(conn), nil
}

func (s *Server) serveWebTransport(w http.ResponseWriter, r *http.Request) (adapters.SocketAdapter, error) {
	conn, err := s.webTransportServer.Upgrade(w, r)
	if err != nil {
		return nil, err
	}
	return adapters.NewWebTransportAdapter(conn, true), nil
}
