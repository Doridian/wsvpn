package connectors

import (
	"context"
	"crypto/tls"
	"errors"
	"net"

	"github.com/Doridian/wsvpn/shared/sockets/adapters"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/webtransport-go"
)

type WebTransportConnector struct {
}

var _ SocketConnector = &WebTransportConnector{}

func NewWebTransportConnector() *WebTransportConnector {
	return &WebTransportConnector{}
}

type quicDialerHelper struct {
	config SocketConnectorConfig
}

func (d *quicDialerHelper) DialEarly(ctx context.Context, addr string, tlsCfg *tls.Config, cfg *quic.Config) (*quic.Conn, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}

	udpConn, err := net.ListenUDP("udp", nil)
	if err != nil {
		return nil, err
	}

	err = d.config.EnhanceConn(udpConn)
	if err != nil {
		_ = udpConn.Close()
		return nil, err
	}

	return quic.DialEarly(ctx, udpConn, udpAddr, tlsCfg, cfg)
}

func (c *WebTransportConnector) Dial(config SocketConnectorConfig) (adapters.SocketAdapter, error) {
	serverURL := *config.GetServerURL()
	serverURL.Scheme = "https"

	if config.GetProxyURL() != nil {
		return nil, errors.New("proxy is not supported for WebTransport at the moment")
	}

	quicDialerInst := &quicDialerHelper{
		config: config,
	}
	dialer := &webtransport.Dialer{
		DialAddr:             quicDialerInst.DialEarly,
		TLSClientConfig:      config.GetTLSConfig(),
		ApplicationProtocols: []string{"wsvpn"},
		QUICConfig: &quic.Config{
			EnableStreamResetPartialDelivery: true,
			EnableDatagrams:                  true,
		},
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
