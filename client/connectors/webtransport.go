package connectors

import (
	"context"
	"errors"

	"github.com/Doridian/wsvpn/shared/sockets/adapters"
	"github.com/lucas-clemente/quic-go/http3"
	"github.com/marten-seemann/webtransport-go"
)

type WebTransportConnector struct {
}

var _ SocketConnector = &WebTransportConnector{}

func NewWebTransportConnector() *WebTransportConnector {
	return &WebTransportConnector{}
}

func (c *WebTransportConnector) Dial(config SocketConnectorConfig) (adapters.SocketAdapter, error) {
	serverURL := *config.GetServerURL()
	serverURL.Scheme = "https"

	var dialer webtransport.Dialer
	if dialer.RoundTripper == nil {
		dialer.RoundTripper = &http3.RoundTripper{}
	}
	dialer.TLSClientConfig = config.GetTLSConfig()

	if config.GetProxyURL() != nil {
		return nil, errors.New("proxy is not support for WebTransport at the moment")
	}

	headers := config.GetHeaders()
	addSupportedSerializationHeader(headers)
	resp, conn, err := dialer.Dial(context.Background(), serverURL.String(), headers)
	if err != nil {
		return nil, err
	}

	serializationType := readSerializationType(resp.Header)
	return adapters.NewWebTransportAdapter(conn, serializationType, false), nil
}

func (c *WebTransportConnector) GetSchemes() []string {
	return []string{"webtransport"}
}
