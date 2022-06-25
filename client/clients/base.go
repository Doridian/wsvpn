package clients

import (
	"crypto/tls"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/Doridian/wsvpn/shared"
	"github.com/Doridian/wsvpn/shared/sockets"
	"github.com/Doridian/wsvpn/shared/sockets/adapters"
	"github.com/songgao/water"
)

const clientEventUp = "up"
const clientEventDown = "down"

type Client struct {
	TLSConfig          *tls.Config
	ProxyUrl           *url.URL
	ServerUrl          *url.URL
	Headers            http.Header
	InterfaceName      string
	SetDefaultGateway  bool
	UpScript           string
	DownScript         string
	SocketConfigurator sockets.SocketConfigurator

	log        *log.Logger
	clientID   string
	serverID   string
	mtu        int
	doIpConfig bool
	iface      *water.Interface
	remoteNet  *shared.VPNNet
	socket     *sockets.Socket
	adapter    adapters.SocketAdapter
}

func NewClient() *Client {
	return &Client{
		Headers:  make(http.Header),
		clientID: "UNSET",
		serverID: "UNSET",
		log:      shared.MakeLogger("CLIENT", "UNSET"),
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
	if c.SocketConfigurator != nil {
		err := c.SocketConfigurator.ConfigureSocket(c.socket)
		if err != nil {
			return err
		}
	}
	c.registerCommandHandlers()

	c.socket.Serve()

	return nil
}

func (c *Client) Wait() {
	c.socket.Wait()
}

func (c *Client) Close() {
	c.runEventScript(clientEventDown)
	if c.iface != nil {
		c.iface.Close()
	}
	if c.socket != nil {
		c.socket.Close()
	}
	if c.adapter != nil {
		c.adapter.Close()
	}
}
