package main

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"flag"
	"io/ioutil"
	"strings"

	"github.com/Doridian/wsvpn/server/authenticators"
	"github.com/Doridian/wsvpn/server/macswitch"
	"github.com/Doridian/wsvpn/server/servers"
	"github.com/Doridian/wsvpn/shared"
	"github.com/Doridian/wsvpn/shared/cli"
	"github.com/google/uuid"
)

var configPtr = flag.String("config", "server.yml", "Config file name")
var printDefaultConfigPtr = flag.Bool("print-default-config", false, "Print default config to console")

func main() {
	flag.Usage = cli.UsageWithVersion
	flag.Parse()

	if *printDefaultConfigPtr {
		print(GetDefaultConfig())
		return
	}

	shared.PrintVersion()

	config := Load(*configPtr)

	var err error
	server := servers.NewServer()

	serverUUID, err := uuid.NewRandom()
	if err != nil {
		panic(err)
	}
	server.SetServerID(serverUUID.String())

	server.VPNNet, err = shared.ParseVPNNet(config.Tunnel.Subnet)
	if err != nil {
		panic(err)
	}

	server.InterfacesConfig = &config.Interfaces
	server.SocketConfigurator = &cli.PingFlagsSocketConfigurator{
		Config: &config.Tunnel.Ping,
	}
	server.DoLocalIpConfig = config.Tunnel.IpConfig.Local
	server.DoRemoteIpConfig = config.Tunnel.IpConfig.Remote
	server.ListenAddr = config.Server.Listen
	server.SetMTU(config.Tunnel.Mtu)
	server.HTTP3Enabled = config.Server.EnableHTTP3

	if strings.ToUpper(config.Tunnel.Mode) == "TAP" {
		macSwitch := macswitch.MakeMACSwitch()
		macSwitch.AllowClientToClient = config.Tunnel.AllowClientToClient
		server.Mode = shared.VPN_MODE_TAP
		server.PacketHandler = macSwitch
	} else {
		server.Mode = shared.VPN_MODE_TUN
	}

	if config.Server.Authenticator.Type == "allow-all" {
		server.Authenticator = &authenticators.AllowAllAuthenticator{}
	} else if config.Server.Authenticator.Type == "htpasswd" {
		server.Authenticator = &authenticators.HtpasswdAuthenticator{}
	} else {
		panic(errors.New("invalid authenticator selected"))
	}

	err = server.Authenticator.Load(config.Server.Authenticator.Config)
	if err != nil {
		panic(err)
	}

	if config.Server.Tls.Certificate != "" || config.Server.Tls.Key != "" || config.Server.Tls.ClientCa != "" {
		if config.Server.Tls.Certificate == "" && config.Server.Tls.Key == "" {
			panic(errors.New("tls-client-ca requires tls-key and tls-cert"))
		}

		if config.Server.Tls.Certificate == "" || config.Server.Tls.Key == "" {
			panic(errors.New("provide either both tls-key and tls-cert or neither"))
		}

		tlsConfig := &tls.Config{}

		cert, err := tls.LoadX509KeyPair(config.Server.Tls.Certificate, config.Server.Tls.Key)
		if err != nil {
			panic(err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}

		if config.Server.Tls.ClientCa != "" {
			tlsClientCAPEM, err := ioutil.ReadFile(config.Server.Tls.ClientCa)
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

		cli.TlsUseConfig(tlsConfig, &config.Server.Tls.Config)

		server.TLSConfig = tlsConfig
	}

	err = server.Serve()
	if err != nil {
		panic(err)
	}
}
