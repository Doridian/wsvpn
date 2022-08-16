package clients

import (
	"crypto/tls"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Doridian/wsvpn/client/connectors"
	"github.com/Doridian/wsvpn/shared"
	"github.com/Doridian/wsvpn/shared/features"
	"github.com/Doridian/wsvpn/shared/sockets"
	"github.com/Doridian/wsvpn/shared/sockets/adapters"
	"github.com/songgao/water"
)

type Client struct {
	shared.EventConfigHolder

	TLSConfig          *tls.Config
	ProxyUrl           *url.URL
	ServerUrl          *url.URL
	Headers            http.Header
	SetDefaultGateway  bool
	SocketConfigurator sockets.SocketConfigurator
	InterfaceConfig    *shared.InterfaceConfig
	AutoReconnectDelay time.Duration

	log        *log.Logger
	clientID   string
	serverID   string
	mtu        int
	doIpConfig bool
	iface      *water.Interface
	remoteNet  *shared.VPNNet
	socket     *sockets.Socket
	adapter    adapters.SocketAdapter
	connectors map[string]connectors.SocketConnector

	sentUpEvent bool

	localFeatures map[features.Feature]bool
}

func NewClient() *Client {
	return &Client{
		Headers:       make(http.Header),
		TLSConfig:     &tls.Config{},
		log:           shared.MakeLogger("CLIENT", ""),
		connectors:    make(map[string]connectors.SocketConnector),
		localFeatures: make(map[features.Feature]bool),
	}
}

func (c *Client) ServeLoop() {
	for {
		c.closeInternal()
		err := c.Serve()
		if err != nil {
			c.log.Printf("Client error: %v", err)
		}
		c.Wait()

		if c.AutoReconnectDelay == time.Duration(0) {
			c.log.Printf("Automatic reconnection disabled, exiting!")
			break
		}
		c.log.Printf("Waiting %s to reconnect...", c.AutoReconnectDelay)
		time.Sleep(c.AutoReconnectDelay)
		c.log.Printf("Reconnecting now!")
	}
}

func (c *Client) Serve() error {
	if c.TLSConfig != nil && c.TLSConfig.InsecureSkipVerify {
		c.log.Printf("WARNING: TLS verification disabled! This can cause security issues!")
	}

	useMTLS := c.TLSConfig != nil && len(c.TLSConfig.Certificates) > 0
	useHTTPAuth := c.Headers.Get("Authorization") != ""

	authentications := make([]string, 0)
	if useMTLS {
		authentications = append(authentications, "mTLS")
	}
	if useHTTPAuth {
		authentications = append(authentications, "HTTP")
	}

	isWarning := true
	authText := "NONE"
	if len(authentications) > 0 {
		authText = strings.Join(authentications, ", ")
		isWarning = false
	}

	c.log.Printf("%sConnecting to %s with authentications: %s", shared.BoolIfString(isWarning, "WARNING: "), c.ServerUrl.Redacted(), authText)

	err := c.connectAdapter()
	if err != nil {
		return err
	}

	c.socket = sockets.MakeSocket(c.log, c.adapter, nil, true)
	err = c.UpdateSocketConfig()
	if err != nil {
		return err
	}
	c.registerCommandHandlers()

	for feat, en := range c.localFeatures {
		c.socket.SetLocalFeature(feat, en)
	}

	c.socket.Serve()

	return nil
}

func (c *Client) UpdateSocketConfig() error {
	if c.socket == nil || c.SocketConfigurator == nil {
		return nil
	}

	return c.SocketConfigurator.ConfigureSocket(c.socket)
}

func (c *Client) Wait() {
	if c.socket == nil {
		return
	}
	c.socket.Wait()
}

func (c *Client) Close() {
	c.AutoReconnectDelay = time.Duration(0)
	c.closeInternal()
}

func (c *Client) closeInternal() {
	if c.sentUpEvent {
		c.doRunEventScript(shared.EventDown)
		c.sentUpEvent = false
	}

	if c.iface != nil {
		c.iface.Close()
		c.iface = nil
	}
	if c.socket != nil {
		c.socket.Close()
		c.socket = nil
	}
	if c.adapter != nil {
		c.adapter.Close()
		c.adapter = nil
	}
}

func (c *Client) doRunEventScript(event string) {
	ifaceName := ""
	if c.iface != nil {
		ifaceName = c.iface.Name()
	}
	remoteNetStr := ""
	if c.remoteNet != nil {
		remoteNetStr = c.remoteNet.GetRaw()
	}

	go func() {
		err := c.RunEventScript(event, remoteNetStr, ifaceName)
		if err != nil {
			c.log.Printf("Error running %s script: %v", event, err)
		}
	}()
}

func (c *Client) SetLocalFeature(feature features.Feature, enabled bool) {
	if !enabled {
		delete(c.localFeatures, feature)
		return
	}
	c.localFeatures[feature] = true
}
