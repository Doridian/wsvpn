package main

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"flag"
	"io/ioutil"

	"github.com/Doridian/wsvpn/server/authenticators"
	"github.com/Doridian/wsvpn/server/macswitch"
	"github.com/Doridian/wsvpn/server/servers"
	"github.com/Doridian/wsvpn/shared"
	"github.com/Doridian/wsvpn/shared/cli"
	"github.com/google/uuid"
)

var mtu = flag.Int("mtu", 1280, "MTU for the tunnel")
var subnetStr = flag.String("subnet", "192.168.3.0/24", "Subnet for the tunnel clients")
var listenAddr = flag.String("listen", "127.0.0.1:9000", "Listen address for the WebSocket interface")
var listenHTTP3Enable = flag.Bool("enable-http3", false, "Enable HTTP/3 protocol")

var authenticatorStrPtr = flag.String("authenticator", "allow-all", "Which authenticator to use (allow-all, htpasswd)")
var authenticatorConfigStrPtr = flag.String("authenticator-config", "", "Authenticator config file (ex. htpasswd file for htpasswd authenticator, empty for default)")

var useTap = flag.Bool("tap", false, "Use a TAP and not a TUN")
var allowClientToClient = flag.Bool("allow-client-to-client", false, "Allow client-to-client communication (in TAP)")
var skipRemoteIpConf = flag.Bool("iface-remote-noconf", false, "Do not send IP or route configuration to remote except MTU")
var skipLocalIpConf = flag.Bool("iface-local-noconf", false, "Do not configure local interface at all except MTU")

var tlsCert = flag.String("tls-cert", "", "TLS certificate file for listener")
var tlsKey = flag.String("tls-key", "", "TLS key file for listener")
var tlsClientCA = flag.String("tls-client-ca", "", "If set, performs TLS client certificate authentication based on given CA certificate")

func main() {
	flag.Usage = cli.UsageWithVersion
	flag.Parse()

	shared.PrintVersion()

	var err error
	server := servers.NewServer()

	serverUUID, err := uuid.NewRandom()
	if err != nil {
		panic(err)
	}
	server.SetServerID(serverUUID.String())

	server.VPNNet, err = shared.ParseVPNNet(*subnetStr)
	if err != nil {
		panic(err)
	}

	server.SocketConfigurator = &cli.PingFlagsSocketConfigurator{}
	server.DoLocalIpConfig = !*skipLocalIpConf
	server.DoRemoteIpConfig = !*skipRemoteIpConf
	server.ListenAddr = *listenAddr
	server.SetMTU(*mtu)
	server.HTTP3Enabled = *listenHTTP3Enable

	if *useTap {
		macSwitch := macswitch.MakeMACSwitch()
		macSwitch.AllowClientToClient = *allowClientToClient
		server.Mode = shared.VPN_MODE_TAP
		server.PacketHandler = macSwitch
	} else {
		server.Mode = shared.VPN_MODE_TUN
	}

	authenticatorStr := *authenticatorStrPtr
	if authenticatorStr == "allow-all" {
		server.Authenticator = &authenticators.AllowAllAuthenticator{}
	} else if authenticatorStr == "htpasswd" {
		server.Authenticator = &authenticators.HtpasswdAuthenticator{}
	} else {
		panic(errors.New("invalid authenticator selected"))
	}

	err = server.Authenticator.Load(*authenticatorConfigStrPtr)
	if err != nil {
		panic(err)
	}

	tlsCertStr := *tlsCert
	tlsKeyStr := *tlsKey
	tlsClientCAStr := *tlsClientCA

	if tlsCertStr != "" || tlsKeyStr != "" || tlsClientCAStr != "" {
		if tlsCertStr == "" && tlsKeyStr == "" {
			panic(errors.New("tls-client-ca requires tls-key and tls-cert"))
		}

		if tlsCertStr == "" || tlsKeyStr == "" {
			panic(errors.New("provide either both tls-key and tls-cert or neither"))
		}

		tlsConfig := &tls.Config{}

		cert, err := tls.LoadX509KeyPair(tlsCertStr, tlsKeyStr)
		if err != nil {
			panic(err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}

		if tlsClientCAStr != "" {
			tlsClientCAPEM, err := ioutil.ReadFile(tlsClientCAStr)
			if err != nil {
				panic(err)
			}

			tlsClientCAPool := x509.NewCertPool()
			ok := tlsClientCAPool.AppendCertsFromPEM(tlsClientCAPEM)
			if !ok {
				panic(errors.New("error reading tls-client-ca PEM"))
			}

			tlsConfig.ClientCAs = tlsClientCAPool
			tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
		}

		cli.TlsUseFlags(tlsConfig)

		server.TLSConfig = tlsConfig
	}

	err = server.Serve()
	if err != nil {
		panic(err)
	}
}
