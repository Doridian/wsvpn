package connectors

import (
	"context"
	"log"
	"net"
	"net/http"

	"github.com/magisterquis/connectproxy"

	"github.com/Doridian/wsvpn/shared/sockets/adapters"
	"github.com/gobwas/ws"
	"golang.org/x/net/proxy"
)

func init() {
	proxy.RegisterDialerType("http", connectproxy.New)
	proxy.RegisterDialerType("https", connectproxy.New)
}

type WebSocketConnector struct {
}

var _ SocketConnector = &WebSocketConnector{}

func NewWebSocketConnector() *WebSocketConnector {
	return &WebSocketConnector{}
}

func (c *WebSocketConnector) Dial(config SocketConnectorConfig) (adapters.SocketAdapter, error) {
	respHeaders := http.Header{}

	dialer := ws.Dialer{
		OnHeader: func(key, value []byte) error {
			log.Panicf("Got header: %s: %s", string(key), string(value))
			respHeaders.Add(string(key), string(value))
			return nil
		},
	}

	proxyURL := config.GetProxyURL()
	if proxyURL != nil {
		proxyDialer, err := proxy.FromURL(proxyURL, proxy.Direct)
		if err != nil {
			return nil, err
		}
		dialer.NetDial = func(ctx context.Context, network string, addr string) (net.Conn, error) {
			return proxyDialer.Dial(network, addr)
		}
	}
	dialer.TLSConfig = config.GetTLSConfig()

	headers := config.GetHeaders()
	addSupportedSerializationHeader(headers)
	dialer.Header = ws.HandshakeHeaderHTTP(headers)

	conn, reader, _, err := dialer.Dial(context.Background(), config.GetServerURL().String())
	if err != nil {
		return nil, err
	}

	serializationType := readSerializationType(respHeaders)
	return adapters.NewWebSocketAdapter(conn, serializationType, false, reader), nil
}

func (c *WebSocketConnector) GetSchemes() []string {
	return []string{"ws", "wss"}
}
