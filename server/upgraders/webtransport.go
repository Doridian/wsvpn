package upgraders

import (
	"crypto/tls"
	"net/http"
	"time"

	"github.com/Doridian/wsvpn/shared/sockets/adapters"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
	"github.com/quic-go/webtransport-go"
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
				Addr:            quicServer.Addr,
				TLSConfig:       quicServer.TLSConfig,
				Handler:         quicServer.Handler,
				EnableDatagrams: true,
				QUICConfig: &quic.Config{
					EnableDatagrams: true,
					KeepAlivePeriod: 10 * time.Second,
				},
			},
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}
}

func (u *WebTransportUpgrader) SetHeaders(headers http.Header) {
	// Nothing to do here
}

func (u *WebTransportUpgrader) SetQuicHeaders(header http.Header) error {
	return u.server.H3.SetQuicHeaders(header)
}

func (u *WebTransportUpgrader) Upgrade(w http.ResponseWriter, r *http.Request) (adapters.SocketAdapter, error) {
	serializationType := handleHTTPSerializationHeaders(w, r)

	conn, err := u.server.Upgrade(w, r)
	if err != nil {
		return nil, err
	}

	return adapters.NewWebTransportAdapter(conn, nil, serializationType, true), nil
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
