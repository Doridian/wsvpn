package cli

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/Doridian/wsvpn/client/clients"
	"github.com/Doridian/wsvpn/shared"
	"github.com/Doridian/wsvpn/shared/cli"
	"github.com/Doridian/wsvpn/shared/features"
	"github.com/Doridian/wsvpn/shared/iface"
)

func reloadConfig(configPtr *string, client *clients.Client) error {
	config, err := Load(*configPtr)
	if err != nil {
		return err
	}

	dest, err := url.Parse(config.Client.Server)
	if err != nil {
		return err
	}

	var userInfo *url.Userinfo
	if config.Client.AuthFile != "" {
		authData, err := os.ReadFile(config.Client.AuthFile)
		if err != nil {
			return err
		}
		authDataStr := strings.Trim(string(authData), "\r\n\t ")
		authDataSplit := strings.SplitN(authDataStr, ":", 2)

		if len(authDataSplit) > 1 {
			userInfo = url.UserPassword(authDataSplit[0], authDataSplit[1])
		} else {
			userInfo = url.User(authDataSplit[0])
		}
	} else {
		userInfo = dest.User
		dest.User = nil
	}

	client.SetBasicAuthFromUserInfo(userInfo)

	client.TLSConfig.InsecureSkipVerify = config.Client.Tls.Config.Insecure
	client.TLSConfig.ServerName = config.Client.Tls.ServerName
	err = cli.TlsUseConfig(client.TLSConfig, &config.Client.Tls.Config)
	if err != nil {
		return err
	}

	if config.Client.Tls.Ca != "" {
		data, err := os.ReadFile(config.Client.Tls.Ca)
		if err != nil {
			return err
		}
		certPool := x509.NewCertPool()
		ok := certPool.AppendCertsFromPEM(data)
		if !ok {
			return errors.New("error loading root CA file")
		}
		client.TLSConfig.RootCAs = certPool
	}

	if config.Client.Tls.Certificate != "" || config.Client.Tls.Key != "" {
		if config.Client.Tls.Certificate == "" || config.Client.Tls.Key == "" {
			return errors.New("provide either both tls.key and tls.certificate or neither")
		}

		tlsClientCertX509, err := tls.LoadX509KeyPair(config.Client.Tls.Certificate, config.Client.Tls.Key)
		if err != nil {
			return err
		}
		client.TLSConfig.Certificates = []tls.Certificate{tlsClientCertX509}
	}

	if config.Client.Proxy != "" {
		proxyUrl, err := url.Parse(config.Client.Proxy)
		if err != nil {
			return err
		}
		client.ProxyUrl = proxyUrl
	}

	client.SocketConfigurator = &cli.PingFlagsSocketConfigurator{
		Config: &config.Tunnel.Ping,
	}
	for feat, en := range config.Tunnel.Features {
		if !features.IsFeatureSupported(feat) {
			return fmt.Errorf("unknown feature: %s", feat)
		}
		client.SetLocalFeature(feat, en)
	}
	client.SetDefaultGateway = config.Tunnel.SetDefaultGateway
	client.ServerUrl = dest
	client.InterfaceConfig = &config.Interface
	client.AutoReconnectDelay = config.Client.AutoReconnectDelay
	client.LoadEventConfig(&config.Scripts)

	return client.UpdateSocketConfig()
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

	client := clients.NewClient()

	defer client.Close()
	cli.RegisterShutdownSignals(func() {
		client.Close()
		os.Exit(0)
	})
	client.RegisterDefaultConnectors()

	err = reloadConfig(configPtr, client)
	if err != nil {
		panic(err)
	}

	reloadSig := make(chan os.Signal, 1)
	signal.Notify(reloadSig, syscall.SIGHUP)
	go func() {
		for {
			<-reloadSig
			log.Printf("Reloading configuration, might not take effect until next connection...")
			err := reloadConfig(configPtr, client)
			if err != nil {
				log.Printf("Error reloading config: %v", err)
			}
		}
	}()

	client.ServeLoop()
}
