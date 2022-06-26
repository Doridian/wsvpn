package clients

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/Doridian/wsvpn/shared"
	"github.com/Doridian/wsvpn/shared/sockets/connectors"
)

func (c *Client) GetProxyUrl() *url.URL {
	return c.ProxyUrl
}

func (c *Client) GetTLSConfig() *tls.Config {
	return c.TLSConfig
}

func (c *Client) GetHeaders() http.Header {
	return c.Headers
}

func (c *Client) GetServerUrl() *url.URL {
	return c.ServerUrl
}

func (c *Client) RegisterDefaultConnectors() {
	c.registerConnector(connectors.NewWebSocketConnector())
	c.registerConnector(connectors.NewWebTransportConnector())
}

func (c *Client) registerConnector(connector connectors.SocketConnector) {
	for _, scheme := range connector.GetSchemes() {
		c.connectors[scheme] = connector
	}
}

func (c *Client) connectAdapter() error {
	scheme := strings.ToLower(c.ServerUrl.Scheme)
	connector, ok := c.connectors[scheme]
	if !ok {
		return fmt.Errorf("invalid protocol: %s", scheme)
	}

	adapter, err := connector.Dial(c)
	if err != nil {
		return err
	}
	c.adapter = adapter

	tlsConnState, ok := c.adapter.GetTLSConnectionState()
	if ok {
		c.log.Printf("TLS %s %s connection established with cipher=%s", shared.TlsVersionString(tlsConnState.Version), c.adapter.Name(), tls.CipherSuiteName(tlsConnState.CipherSuite))
	} else {
		c.log.Printf("Unencrypted %s connection established", c.adapter.Name())
	}

	return nil
}
