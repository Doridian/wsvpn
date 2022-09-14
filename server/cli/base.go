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
	"github.com/Doridian/wsvpn/shared/iface"
	"github.com/google/uuid"
)

var tlsConfig *tls.Config

func getTLSConfig(_ *tls.ClientHelloInfo) (*tls.Config, error) {
	return tlsConfig, nil
}

func getTLSCert(_ *tls.ClientHelloInfo) (*tls.Certificate, error) {
	return &tlsConfig.Certificates[0], nil
}

func reloadConfig(configPtr *string, server *servers.Server, initialConfig bool) error {
	config, err := Load(*configPtr)
	if err != nil {
		return err
	}

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
	server.DoLocalIPConfig = config.Tunnel.IPConfig.Local
	server.DoRemoteIPConfig = config.Tunnel.IPConfig.Remote
	for feat, en := range config.Tunnel.Features {
		if !features.IsFeatureSupported(feat) {
			return fmt.Errorf("unknown feature: %s", feat)
		}
		server.SetLocalFeature(feat, en)
	}
	server.LoadEventConfig(&config.Scripts)

	var vpnMode shared.VPNMode
	switch strings.ToUpper(config.Tunnel.Mode) {
	case "TAP":
		vpnMode = shared.VPNModeTAP
	case "TUN":
		vpnMode = shared.VPNModeTUN
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

	err = server.SetMTU(config.Tunnel.MTU)
	if err != nil {
		return err
	}

	server.MaxConnectionsPerUser = config.Server.MaxConnectionsPerUser
	switch config.Server.MaxConnectionsPerUserMode {
	case "kill-oldest":
		server.MaxConnectionsPerUserMode = servers.MaxConnectionsPerUserKillOldest
	case "prevent-new":
		server.MaxConnectionsPerUserMode = servers.MaxConnectionsPerUserPreventNew
	}

	if !initialConfig && server.InterfaceConfig.OneInterfacePerConnection != config.Interface.OneInterfacePerConnection {
		log.Printf("WARNING: Ignroing interface config due to change of interface.one-interface-per-connection on reload")
	} else {
		server.InterfaceConfig = &config.Interface

		if !server.InterfaceConfig.OneInterfacePerConnection {
			if server.Mode == shared.VPNModeTAP {
				var macSwitch *macswitch.MACSwitch
				if initialConfig {
					macSwitch = macswitch.MakeMACSwitch()
					server.PacketHandler = macSwitch
				} else {
					macSwitch = server.PacketHandler.(*macswitch.MACSwitch)
				}
				macSwitch.AllowClientToClient = config.Tunnel.AllowClientToClient
				macSwitch.AllowIPSpoofing = config.Tunnel.AllowIPSpoofing
				macSwitch.AllowUnknownEtherTypes = config.Tunnel.AllowUnknownEtherTypes
				macSwitch.AllowMACChanging = config.Tunnel.AllowMACChanging
				macSwitch.AllowedMACsPerConnection = config.Tunnel.AllowedMACsPerConnection
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

	if config.Server.TLS.Certificate != "" || config.Server.TLS.Key != "" || config.Server.TLS.ClientCA != "" {
		if config.Server.TLS.Certificate == "" && config.Server.TLS.Key == "" {
			return errors.New("tls-client-ca requires tls-key and tls-cert")
		}

		if config.Server.TLS.Certificate == "" || config.Server.TLS.Key == "" {
			return errors.New("provide either both tls-key and tls-cert or neither")
		}

		newTLSConfig := &tls.Config{}

		cert, err := tls.LoadX509KeyPair(config.Server.TLS.Certificate, config.Server.TLS.Key)
		if err != nil {
			return err
		}
		newTLSConfig.Certificates = []tls.Certificate{cert}

		if config.Server.TLS.ClientCA != "" {
			var tlsClientCAPEM []byte
			tlsClientCAPEM, err = os.ReadFile(config.Server.TLS.ClientCA)
			if err != nil {
				return err
			}

			tlsClientCAPool := x509.NewCertPool()
			ok := tlsClientCAPool.AppendCertsFromPEM(tlsClientCAPEM)
			if !ok {
				return errors.New("error reading tls-client-ca PEM")
			}

			newTLSConfig.ClientCAs = tlsClientCAPool
			newTLSConfig.ClientAuth = tls.RequireAndVerifyClientCert
		}

		err = cli.TLSUseConfig(newTLSConfig, &config.Server.TLS.Config)
		if err != nil {
			return err
		}

		tlsConfig = newTLSConfig

		if server.TLSConfig == nil {
			if initialConfig {
				server.TLSConfig = &tls.Config{
					GetConfigForClient: getTLSConfig,
					GetCertificate:     getTLSCert,
				}
			} else {
				log.Printf("WARNING: Ignoring enablement of TLS on reload")
			}
		}
	} else if !initialConfig && server.TLSConfig != nil {
		log.Printf("WARNING: Ignoring disablement of TLS on reload")
	}

	return server.UpdateSocketConfig()
}

func Main(configPtr *string, printDefaultConfigPtr *bool) {
	if *printDefaultConfigPtr {
		fmt.Println(GetDefaultConfig())
		return
	}

	shared.PrintVersion()

	err := iface.InitializeWater()
	if err != nil {
		log.Printf("Could not initialize network interface library (this may cause crashes): %v", err)
	}

	server := servers.NewServer()

	cli.RegisterShutdownSignals(func() {
		server.Close()
		os.Exit(0)
	})
	defer server.Close()

	serverUUID, err := uuid.NewRandom()
	if err != nil {
		panic(err)
	}
	server.SetServerID(serverUUID.String())

	err = reloadConfig(configPtr, server, true)
	if err != nil {
		panic(err)
	}

	reloadSig := make(chan os.Signal, 1)
	signal.Notify(reloadSig, syscall.SIGHUP)
	go func() {
		for {
			<-reloadSig
			log.Printf("Reloading configuration, might not take effect until next connection...")
			reloadErr := reloadConfig(configPtr, server, false)
			if reloadErr != nil {
				log.Printf("Error reloading config: %v", reloadErr)
			}
		}
	}()

	err = server.Serve()
	if err != nil {
		panic(err)
	}
}
