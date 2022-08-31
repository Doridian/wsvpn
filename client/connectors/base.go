package connectors

import (
	"crypto/tls"
	"net/http"
	"net/url"

	"github.com/Doridian/wsvpn/shared/sockets/adapters"
)

type SocketConnector interface {
	Dial(config SocketConnectorConfig) (adapters.SocketAdapter, error)
	GetSchemes() []string
}

type SocketConnectorConfig interface {
	GetProxyURL() *url.URL
	GetTLSConfig() *tls.Config
	GetHeaders() http.Header
	GetServerURL() *url.URL
}
