package upgraders

import (
	"io"
	"net/http"

	"github.com/Doridian/wsvpn/shared/sockets/adapters"
)

type SocketUpgrader interface {
	io.Closer

	SetHeaders(headers http.Header)
	Upgrade(w http.ResponseWriter, r *http.Request) (adapters.SocketAdapter, error)
	// Listen binds any listener the upgrader needs on its own, Serve then
	// blocks serving it. Splitting the two lets the server only report itself
	// as online once every socket is actually bound.
	Listen() error
	Serve() error
	Matches(r *http.Request) bool
}
