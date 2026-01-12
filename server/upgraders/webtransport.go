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
	tlsConfig := quicServer.TLSConfig.Clone()
	tlsConfig.NextProtos = []string{http3.NextProtoH3}
	if tlsConfig.GetConfigForClient != nil {
		oldConfig := tlsConfig.GetConfigForClient
		tlsConfig.GetConfigForClient = func(chi *tls.ClientHelloInfo) (*tls.Config, error) {
			cfg, err := oldConfig(chi)
			if err != nil {
				return nil, err
			}
			cfg = cfg.Clone()
			cfg.NextProtos = []string{http3.NextProtoH3}
			return cfg, nil
		}
	}

	upgrader := &WebTransportUpgrader{
		server: &webtransport.Server{
			ApplicationProtocols: []string{"wsvpn"},
			H3: &http3.Server{
				Addr:            quicServer.Addr,
				TLSConfig:       tlsConfig,
				Handler:         quicServer.Handler,
				EnableDatagrams: true,
				QUICConfig: &quic.Config{
					EnableStreamResetPartialDelivery: true,
					EnableDatagrams:                  true,
					KeepAlivePeriod:                  10 * time.Second,
				},
			},
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}
	webtransport.ConfigureHTTP3Server(upgrader.server.H3)
	return upgrader
}

func (u *WebTransportUpgrader) SetHeaders(headers http.Header) {
	// Nothing to do here
}

func (u *WebTransportUpgrader) SetQUICHeaders(header http.Header) error {
	return u.server.H3.SetQUICHeaders(header)
}

func (u *WebTransportUpgrader) Upgrade(w http.ResponseWriter, r *http.Request) (adapters.SocketAdapter, error) {
	serializationType := handleHTTPSerializationHeaders(w, r)

	conn, err := u.server.Upgrade(w, r)
	if err != nil {
		return nil, err
	}

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
