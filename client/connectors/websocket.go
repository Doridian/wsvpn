package connectors

import (
	"net/http"
	"net/url"

	"github.com/Doridian/wsvpn/shared/sockets/adapters"
	"github.com/gorilla/websocket"
)

type WebSocketConnector struct {
}

var _ SocketConnector = &WebSocketConnector{}

func NewWebSocketConnector() *WebSocketConnector {
	return &WebSocketConnector{}
}

func (c *WebSocketConnector) Dial(config SocketConnectorConfig) (adapters.SocketAdapter, error) {
	dialer := websocket.Dialer{}

	proxyUrl := config.GetProxyUrl()
	if proxyUrl != nil {
		dialer.Proxy = func(_ *http.Request) (*url.URL, error) {
			return proxyUrl, nil
		}
	}
	dialer.TLSClientConfig = config.GetTLSConfig()

	conn, _, err := dialer.Dial(config.GetServerUrl().String(), config.GetHeaders())
	if err != nil {
		return nil, err
	}

	return adapters.NewWebSocketAdapter(conn), nil
}

func (c *WebSocketConnector) GetSchemes() []string {
	return []string{"ws", "wss"}
}
