package cli

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"log"
	"net/http"
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
		authData, authErr := os.ReadFile(config.Client.AuthFile)
		if authErr != nil {
			return authErr
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

	client.Headers = http.Header{}
	for name, values := range config.Client.Headers {
		for _, value := range values {
			client.Headers.Add(name, value)
		}
	}

	client.SetBasicAuthFromUserInfo(userInfo)

	if client.Headers.Get("User-Agent") == "" {
		client.Headers.Set("User-Agent", fmt.Sprintf("wsvpn/%s", shared.Version))
	}

	client.TLSConfig.InsecureSkipVerify = config.Client.TLS.Config.Insecure
	client.TLSConfig.ServerName = config.Client.TLS.ServerName

	err = cli.TLSUseConfig(client.TLSConfig, &config.Client.TLS.Config)
	if err != nil {
		return err
	}

	if config.Client.TLS.CA != "" {
		data, err := os.ReadFile(config.Client.TLS.CA)
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

	if config.Client.TLS.Certificate != "" || config.Client.TLS.Key != "" {
		if config.Client.TLS.Certificate == "" || config.Client.TLS.Key == "" {
			return errors.New("provide either both tls.key and tls.certificate or neither")
		}

		tlsClientCertX509, err := tls.LoadX509KeyPair(config.Client.TLS.Certificate, config.Client.TLS.Key)
		if err != nil {
			return err
		}
		client.TLSConfig.Certificates = []tls.Certificate{tlsClientCertX509}
	}

	if config.Client.Proxy != "" {
		proxyURL, err := url.Parse(config.Client.Proxy)
		if err != nil {
			return err
		}
		client.ProxyURL = proxyURL
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
	client.ServerURL = dest
	client.InterfaceConfig = &config.Interface
	client.InterfaceConfig.OneInterfacePerConnection = false
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
