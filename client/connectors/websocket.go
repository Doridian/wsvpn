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

	proxyURL := config.GetProxyURL()
	if proxyURL != nil {
		dialer.Proxy = func(_ *http.Request) (*url.URL, error) {
			return proxyURL, nil
		}
	}
	dialer.TLSClientConfig = config.GetTLSConfig()

	headers := config.GetHeaders()
	addSupportedSerializationHeader(headers)
	conn, resp, err := dialer.Dial(config.GetServerURL().String(), headers)
	if err != nil {
		return nil, err
	}

	serializationType := readSerializationType(resp.Header)
	return adapters.NewWebSocketAdapter(conn, serializationType, true), nil
}

func (c *WebSocketConnector) GetSchemes() []string {
	return []string{"ws", "wss"}
}
