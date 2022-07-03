package upgraders

import (
	"net/http"

	"github.com/Doridian/wsvpn/shared/sockets/adapters"
	"github.com/lucas-clemente/quic-go/http3"
	"github.com/marten-seemann/webtransport-go"
)

type WebTransportUpgrader struct {
	server *webtransport.Server
}

var _ SocketUpgrader = &WebTransportUpgrader{}

func NewWebTransportUpgrader(quicServer *http3.Server) *WebTransportUpgrader {
	return &WebTransportUpgrader{
		server: &webtransport.Server{
			H3:          *quicServer,
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}
}

func (u *WebTransportUpgrader) Upgrade(w http.ResponseWriter, r *http.Request) (adapters.SocketAdapter, error) {
	conn, err := u.server.Upgrade(w, r)
	if err != nil {
		return nil, err
	}

	serializationType := determineBestSerialization(r.Header)
	return adapters.NewWebTransportAdapter(conn, serializationType, true), nil
}

func (u *WebTransportUpgrader) ListenAndServe() error {
	return u.server.ListenAndServe()
}

func (u *WebTransportUpgrader) Matches(r *http.Request) bool {
	return r.Proto == "webtransport"
}

func (u *WebTransportUpgrader) Close() error {
	return u.server.Close()
}
