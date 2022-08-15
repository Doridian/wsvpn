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

	newVPNNet, err := shared.ParseVPNNet(config.Tunnel.Subnet)
	if err != nil {
		return err
	}

	if initialConfig {
		server.VPNNet = newVPNNet
	} else if !server.VPNNet.Equals(newVPNNet) {
		log.Printf("WARNING: Ignoring change of tunnel.subnet on reload")
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

	vpnMode := shared.VPN_MODE_INVALID
	switch strings.ToUpper(config.Tunnel.Mode) {
	case "TAP":
		vpnMode = shared.VPN_MODE_TAP
	case "TUN":
		vpnMode = shared.VPN_MODE_TUN
	default:
		return errors.New("invalid VPN mode selected")
	}

	if initialConfig {
		server.ListenAddr = config.Server.Listen
		server.HTTP3Enabled = config.Server.EnableHTTP3

		server.Mode = vpnMode
	} else {
		if server.ListenAddr != config.Server.Listen {
			log.Printf("WARNING: Ignoring change of server.listen on reload")
		}
		if server.HTTP3Enabled != config.Server.EnableHTTP3 {
			log.Printf("WARNING: Ignoring change of server.enable-http3 on reload")
		}
		if server.Mode != vpnMode {
			log.Printf("WARNING: Ignoring change of tunnel.mode on reload")
		}
	}

	err = server.SetMTU(config.Tunnel.Mtu)
	if err != nil {
		return err
	}

	if !initialConfig && server.InterfaceConfig.OneInterfacePerConnection != config.Interface.OneInterfacePerConnection {
		log.Printf("WARNING: Ignroing interface config due to change of interface.one-interface-per-connection on reload")
	} else {
		server.InterfaceConfig = &config.Interface

		if !server.InterfaceConfig.OneInterfacePerConnection {
			if server.Mode == shared.VPN_MODE_TAP {
				var macSwitch *macswitch.MACSwitch
				if initialConfig {
					macSwitch = macswitch.MakeMACSwitch()
					server.PacketHandler = macSwitch
				} else {
					macSwitch = server.PacketHandler.(*macswitch.MACSwitch)
				}
				macSwitch.AllowClientToClient = config.Tunnel.AllowClientToClient
				macSwitch.AllowIpSpoofing = config.Tunnel.AllowIpSpoofing
				macSwitch.AllowUnknownEtherTypes = config.Tunnel.AllowUnknownEtherTypes
				macSwitch.AllowMacChanging = config.Tunnel.AllowMacChanging
				macSwitch.AllowedMacsPerConnection = config.Tunnel.AllowedMacsPerConnection
				macSwitch.ConfigUpdate()
			} else {
				var ipSwitch *ipswitch.IPSwitch
				if initialConfig {
					ipSwitch = ipswitch.MakeIPSwitch()
					server.PacketHandler = ipSwitch
				} else {
					ipSwitch = server.PacketHandler.(*ipswitch.IPSwitch)
				}
				ipSwitch.AllowClientToClient = config.Tunnel.AllowClientToClient
			}
		}
	}

	var newAuthenticator authenticators.Authenticator

	switch strings.ToLower(config.Server.Authenticator.Type) {
	case "allow-all":
		newAuthenticator = &authenticators.AllowAllAuthenticator{}
	case "htpasswd":
		newAuthenticator = &authenticators.HtpasswdAuthenticator{}
	default:
		return errors.New("invalid authenticator selected")
	}

	err = newAuthenticator.Load(config.Server.Authenticator.Config)
	if err != nil {
		return err
	}

	server.Authenticator = newAuthenticator

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
			if initialConfig {
				server.TLSConfig = &tls.Config{
					GetConfigForClient: getTlsConfig,
					GetCertificate:     getTlsCert,
				}
			} else {
				log.Printf("WARNING: Ignoring enablement of TLS on reload")
			}
		}
	} else if !initialConfig && server.TLSConfig != nil {
		log.Printf("WARNING: Ignoring disablement of TLS on reload")
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
