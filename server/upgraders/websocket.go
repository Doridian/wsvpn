package upgraders

import (
	"net/http"

	"github.com/Doridian/wsvpn/shared/sockets/adapters"
	"github.com/gobwas/ws"
)

type WebSocketUpgrader struct {
	upgrader *ws.HTTPUpgrader
}

var _ SocketUpgrader = &WebSocketUpgrader{}

func NewWebSocketUpgrader() *WebSocketUpgrader {
	return &WebSocketUpgrader{
		upgrader: &ws.HTTPUpgrader{},
	}
}

func (u *WebSocketUpgrader) SetHeaders(headers http.Header) {
	u.upgrader.Header = headers
}

func (u *WebSocketUpgrader) Upgrade(w http.ResponseWriter, r *http.Request) (adapters.SocketAdapter, error) {
	serializationType := handleHTTPSerializationHeaders(w, r)

	conn, _, _, err := u.upgrader.Upgrade(r, w)
	if err != nil {
		return nil, err
	}

	return adapters.NewWebSocketAdapter(conn, serializationType, true, nil), nil
}

func (u *WebSocketUpgrader) ListenAndServe() error {
	return nil
}

func (u *WebSocketUpgrader) Matches(r *http.Request) bool {
	return r.Header.Get("Upgrade") == "websocket"
}

func (u *WebSocketUpgrader) Close() error {
	return nil
}
