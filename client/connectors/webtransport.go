package connectors

import (
	"context"
	"errors"

	"github.com/Doridian/wsvpn/shared/sockets/adapters"
	"github.com/marten-seemann/webtransport-go"
)

type WebTransportConnector struct {
}

var _ SocketConnector = &WebTransportConnector{}

func NewWebTransportConnector() *WebTransportConnector {
	return &WebTransportConnector{}
}

func (c *WebTransportConnector) Dial(config SocketConnectorConfig) (adapters.SocketAdapter, error) {
	serverUrl := *config.GetServerUrl()
	serverUrl.Scheme = "https"
	dialer := webtransport.Dialer{}
	dialer.TLSClientConf = config.GetTLSConfig()

	if config.GetProxyUrl() != nil {
		return nil, errors.New("proxy is not support for WebTransport at the moment")
	}

	headers := config.GetHeaders()
	addSupportedSerializationHeader(headers)
	resp, conn, err := dialer.Dial(context.Background(), serverUrl.String(), headers)
	if err != nil {
		return nil, err
	}

	serializationType := readSerializationType(resp.Header)
	return adapters.NewWebTransportAdapter(conn, serializationType, false), nil
}

func (c *WebTransportConnector) GetSchemes() []string {
	return []string{"webtransport"}
}
