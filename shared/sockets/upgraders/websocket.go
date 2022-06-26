package upgraders

import (
	"net/http"

	"github.com/Doridian/wsvpn/shared/sockets/adapters"
	"github.com/gorilla/websocket"
)

type WebSocketUpgrader struct {
	upgrader *websocket.Upgrader
}

var _ SocketUpgrader = &WebSocketUpgrader{}

func NewWebSocketUpgrader() *WebSocketUpgrader {
	return &WebSocketUpgrader{
		upgrader: &websocket.Upgrader{
			ReadBufferSize:  2048,
			WriteBufferSize: 2048,
			CheckOrigin:     func(r *http.Request) bool { return true },
		},
	}
}

func (u *WebSocketUpgrader) Upgrade(w http.ResponseWriter, r *http.Request) (adapters.SocketAdapter, error) {
	conn, err := u.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, err
	}
	return adapters.NewWebSocketAdapter(conn), nil
}

func (u *WebSocketUpgrader) ListenAndServe() error {
	return nil
}

func (u *WebSocketUpgrader) Matches(r *http.Request) bool {
	return r.Header.Get("Upgrade") == "websocket"
}
