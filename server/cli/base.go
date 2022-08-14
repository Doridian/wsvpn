package cli

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/Doridian/wsvpn/server/authenticators"
	"github.com/Doridian/wsvpn/server/ipswitch"
	"github.com/Doridian/wsvpn/server/macswitch"
	"github.com/Doridian/wsvpn/server/servers"
	"github.com/Doridian/wsvpn/shared"
	"github.com/Doridian/wsvpn/shared/cli"
	"github.com/Doridian/wsvpn/shared/features"
	"github.com/google/uuid"
)

var tlsConfig *tls.Config

func getTlsConfig(_ *tls.ClientHelloInfo) (*tls.Config, error) {
	return tlsConfig, nil
}

func getTlsCert(_ *tls.ClientHelloInfo) (*tls.Certificate, error) {
	return &tlsConfig.Certificates[0], nil
}

func reloadConfig(configPtr *string, server *servers.Server, initialConfig bool) error {
	var err error

	config := Load(*configPtr)

	server.VPNNet, err = shared.ParseVPNNet(config.Tunnel.Subnet)
	if err != nil {
		return err
	}

	server.SocketConfigurator = &cli.PingFlagsSocketConfigurator{
		Config: &config.Tunnel.Ping,
	}
	server.DoLocalIpConfig = config.Tunnel.IpConfig.Local
	server.DoRemoteIpConfig = config.Tunnel.IpConfig.Remote
	for feat, en := range config.Tunnel.Features {
		if !features.IsFeatureSupported(feat) {
			return fmt.Errorf("unknown feature: %s", feat)
		}
		server.SetLocalFeature(feat, en)
	}
	server.LoadEventConfig(&config.Scripts)

	if initialConfig {
		server.InterfaceConfig = &config.Interface
		server.ListenAddr = config.Server.Listen
		server.SetMTU(config.Tunnel.Mtu)
		server.HTTP3Enabled = config.Server.EnableHTTP3

		if strings.ToUpper(config.Tunnel.Mode) == "TAP" {
			server.Mode = shared.VPN_MODE_TAP
		} else {
			server.Mode = shared.VPN_MODE_TUN
		}

		if !config.Interface.OneInterfacePerConnection {
			if server.Mode == shared.VPN_MODE_TAP {
				macSwitch := macswitch.MakeMACSwitch()
				macSwitch.AllowClientToClient = config.Tunnel.AllowClientToClient
				macSwitch.AllowIpSpoofing = config.Tunnel.AllowIpSpoofing
				macSwitch.AllowUnknownEtherTypes = config.Tunnel.AllowUnknownEtherTypes
				server.PacketHandler = macSwitch
			} else {
				ipSwitch := ipswitch.MakeIPSwitch()
				ipSwitch.AllowClientToClient = config.Tunnel.AllowClientToClient
				server.PacketHandler = ipSwitch
			}
		}
	} else {
		log.Printf("NOTE: Can not reload interface section, TUN/TAP mode, MTU, listeners or HTTP/3 state!")
	}

	if config.Server.Authenticator.Type == "allow-all" {
		server.Authenticator = &authenticators.AllowAllAuthenticator{}
	} else if config.Server.Authenticator.Type == "htpasswd" {
		server.Authenticator = &authenticators.HtpasswdAuthenticator{}
	} else {
		return errors.New("invalid authenticator selected")
	}

	err = server.Authenticator.Load(config.Server.Authenticator.Config)
	if err != nil {
		return err
	}

	if config.Server.Tls.Certificate != "" || config.Server.Tls.Key != "" || config.Server.Tls.ClientCa != "" {
		if config.Server.Tls.Certificate == "" && config.Server.Tls.Key == "" {
			return errors.New("tls-client-ca requires tls-key and tls-cert")
		}

		if config.Server.Tls.Certificate == "" || config.Server.Tls.Key == "" {
			return errors.New("provide either both tls-key and tls-cert or neither")
		}

		newTlsConfig := &tls.Config{}

		cert, err := tls.LoadX509KeyPair(config.Server.Tls.Certificate, config.Server.Tls.Key)
		if err != nil {
			return err
		}
		newTlsConfig.Certificates = []tls.Certificate{cert}

		if config.Server.Tls.ClientCa != "" {
			tlsClientCAPEM, err := os.ReadFile(config.Server.Tls.ClientCa)
			if err != nil {
				return err
			}

			tlsClientCAPool := x509.NewCertPool()
			ok := tlsClientCAPool.AppendCertsFromPEM(tlsClientCAPEM)
			if !ok {
				return errors.New("error reading tls-client-ca PEM")
			}

			newTlsConfig.ClientCAs = tlsClientCAPool
			newTlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
		}

		err = cli.TlsUseConfig(newTlsConfig, &config.Server.Tls.Config)
		if err != nil {
			return err
		}

		tlsConfig = newTlsConfig

		if server.TLSConfig == nil {
			if !initialConfig {
				return errors.New("cannot enable TLS while server is already running")
			}
			server.TLSConfig = &tls.Config{
				GetConfigForClient: getTlsConfig,
				GetCertificate:     getTlsCert,
			}
		}
	} else if !initialConfig && server.TLSConfig != nil {
		return errors.New("cannot disable TLS while server is already running")
	}

	return nil
}

func Main(configPtr *string, printDefaultConfigPtr *bool) {
	if *printDefaultConfigPtr {
		fmt.Println(GetDefaultConfig())
		return
	}

	shared.PrintVersion()

	server := servers.NewServer()

	cli.RegisterShutdownSignals(func() {
		server.Close()
		os.Exit(0)
	})

	serverUUID, err := uuid.NewRandom()
	if err != nil {
		panic(err)
	}
	server.SetServerID(serverUUID.String())

	err = reloadConfig(configPtr, server, true)
	if err != nil {
		panic(err)
	}

	runReloadLoop := true
	defer func() {
		runReloadLoop = false
	}()

	reloadSig := make(chan os.Signal, 1)
	signal.Notify(reloadSig, syscall.SIGHUP)
	go func() {
		for runReloadLoop {
			<-reloadSig
			log.Printf("Reloading configuration, might not take effect until next connection...")
			err := reloadConfig(configPtr, server, false)
			if err != nil {
				log.Printf("Error reloading config: %v", err)
			}
		}
	}()

	err = server.Serve()
	if err != nil {
		panic(err)
	}
}
