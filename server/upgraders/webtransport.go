package upgraders

import (
	"crypto/tls"
	"errors"
	"net/http"

	"github.com/Doridian/wsvpn/shared/sockets/adapters"
	"github.com/lucas-clemente/quic-go"
	"github.com/lucas-clemente/quic-go/http3"
	"github.com/marten-seemann/webtransport-go"
)

type WebTransportUpgrader struct {
	server *webtransport.Server
}

type QuicServerConfig struct {
	Addr      string
	TLSConfig *tls.Config
	Handler   http.HandlerFunc
}

var _ SocketUpgrader = &WebTransportUpgrader{}

func NewWebTransportUpgrader(quicServer *QuicServerConfig) *WebTransportUpgrader {
	return &WebTransportUpgrader{
		server: &webtransport.Server{
			H3: http3.Server{
				Addr:      quicServer.Addr,
				TLSConfig: quicServer.TLSConfig,
				Handler:   quicServer.Handler,
			},
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}
}

func (u *WebTransportUpgrader) SetQuicHeaders(header http.Header) error {
	return u.server.H3.SetQuicHeaders(header)
}

func (u *WebTransportUpgrader) Upgrade(w http.ResponseWriter, r *http.Request) (adapters.SocketAdapter, error) {
	serializationType := handleHTTPSerializationHeaders(w, r)

	hijacker, ok := w.(http3.Hijacker)
	if !ok {
		return nil, errors.New("unexpected: ResponseWriter is not http3.Hijacker")
	}
	qconn, ok := hijacker.StreamCreator().(quic.Connection)
	if !ok {
		return nil, errors.New("unexpected: StreamCreator is not quic.Connection")
	}

	conn, err := u.server.Upgrade(w, r)
	if err != nil {
		return nil, err
	}

	return adapters.NewWebTransportAdapter(qconn, conn, serializationType, true), nil
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
