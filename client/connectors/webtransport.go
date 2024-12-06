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

type quicDialer struct {
	transport *quic.Transport
}

func (d *quicDialer) Dial(ctx context.Context, addr string, tlsCfg *tls.Config, cfg *quic.Config) (quic.EarlyConnection, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}
	return d.transport.DialEarly(ctx, udpAddr, tlsCfg, cfg)
}

func (c *WebTransportConnector) Dial(config SocketConnectorConfig) (adapters.SocketAdapter, error) {
	serverURL := *config.GetServerURL()
	serverURL.Scheme = "https"

	if config.GetProxyURL() != nil {
		return nil, errors.New("proxy is not supported for WebTransport at the moment")
	}

	udpConn, err := net.ListenUDP("udp", nil)
	if err != nil {
		return nil, err
	}

	err = config.EnhanceConn(udpConn)
	if err != nil {
		_ = udpConn.Close()
		return nil, err
	}
	quicDialer := &quicDialer{
		transport: &quic.Transport{Conn: udpConn},
	}

	var dialer webtransport.Dialer
	if dialer.QUICConfig == nil {
		dialer.DialAddr = quicDialer.Dial
		dialer.QUICConfig = &quic.Config{
			EnableDatagrams: true,
		}
	}
	dialer.TLSClientConfig = config.GetTLSConfig()

	headers := config.GetHeaders()
	addSupportedSerializationHeader(headers)
	resp, conn, err := dialer.Dial(context.Background(), serverURL.String(), headers)
	if err != nil {
		_ = udpConn.Close()
		return nil, err
	}

	serializationType := readSerializationType(resp.Header)
	return adapters.NewWebTransportAdapter(conn, udpConn, serializationType, false), nil
}

func (c *WebTransportConnector) GetSchemes() []string {
	return []string{"webtransport"}
}
