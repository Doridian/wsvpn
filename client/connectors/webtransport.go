package connectors

import (
	"context"
	"crypto/tls"
	"errors"
	"net"

	"github.com/Doridian/wsvpn/shared/sockets/adapters"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
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
	if dialer.RoundTripper == nil {
		dialer.RoundTripper = &http3.RoundTripper{
			Dial: quicDialer.Dial,
		}
	}
	dialer.TLSClientConfig = config.GetTLSConfig()

	if config.GetProxyURL() != nil {
		return nil, errors.New("proxy is not supported for WebTransport at the moment")
	}

	headers := config.GetHeaders()
	addSupportedSerializationHeader(headers)
	resp, conn, err := dialer.Dial(context.Background(), serverURL.String(), headers)
	if err != nil {
		return nil, err
	}

	hijacker, ok := resp.Body.(http3.Hijacker)
	if !ok {
		return nil, errors.New("unexpected: Body is not http3.Hijacker")
	}
	qconn, ok := hijacker.StreamCreator().(quic.Connection)
	if !ok {
		return nil, errors.New("unexpected: StreamCreator is not quic.Connection")
	}

	serializationType := readSerializationType(resp.Header)
	return adapters.NewWebTransportAdapter(qconn, conn, serializationType, false), nil
}

func (c *WebTransportConnector) GetSchemes() []string {
	return []string{"webtransport"}
}
