package clients

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/Doridian/wsvpn/client/connectors"
	"github.com/Doridian/wsvpn/shared"
	"github.com/Doridian/wsvpn/shared/commands"
)

func (c *Client) GetProxyURL() *url.URL {
	return c.ProxyURL
}

func (c *Client) GetTLSConfig() *tls.Config {
	return c.TLSConfig.Clone()
}

func (c *Client) GetHeaders() http.Header {
	return c.Headers.Clone()
}

func (c *Client) GetServerURL() *url.URL {
	return c.ServerURL
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
	scheme := strings.ToLower(c.ServerURL.Scheme)
	connector, ok := c.connectors[scheme]
	if !ok {
		return fmt.Errorf("invalid protocol: %s", scheme)
	}

	adapter, err := connector.Dial(c)
	if err != nil {
		return err
	}
	c.adapter = adapter

	c.log.Printf("Command serialization: %s", commands.SerializationTypeToString(adapter.GetCommandSerializationType()))

	tlsConnState, ok := c.adapter.GetTLSConnectionState()
	if ok {
		c.log.Printf("TLS %s %s connection established with cipher=%s", shared.TLSVersionString(tlsConnState.Version), c.adapter.Name(), tls.CipherSuiteName(tlsConnState.CipherSuite))
	} else {
		c.log.Printf("Unencrypted %s connection established", c.adapter.Name())
	}

	return nil
}
