package cli

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io/ioutil"
	"net/url"
	"os"
	"strings"

	"github.com/Doridian/wsvpn/client/clients"
	"github.com/Doridian/wsvpn/shared"
	"github.com/Doridian/wsvpn/shared/cli"
)

func Main(configPtr *string, printDefaultConfigPtr *bool) {
	if *printDefaultConfigPtr {
		print(GetDefaultConfig())
		return
	}

	shared.PrintVersion()

	config := Load(*configPtr)

	dest, err := url.Parse(config.Client.Server)
	if err != nil {
		panic(err)
	}

	var userInfo *url.Userinfo

	if config.Client.AuthFile != "" {
		authData, err := ioutil.ReadFile(config.Client.AuthFile)
		if err != nil {
			panic(err)
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
	}

	tlsConfig := &tls.Config{}

	tlsConfig.InsecureSkipVerify = config.Client.Tls.Config.Insecure
	tlsConfig.ServerName = config.Client.Tls.ServerName
	err = cli.TlsUseConfig(tlsConfig, &config.Client.Tls.Config)
	if err != nil {
		panic(err)
	}

	if config.Client.Tls.Ca != "" {
		data, err := ioutil.ReadFile(config.Client.Tls.Ca)
		if err != nil {
			panic(err)
		}
		certPool := x509.NewCertPool()
		ok := certPool.AppendCertsFromPEM(data)
		if !ok {
			panic(errors.New("error loading root CA file"))
		}
		tlsConfig.RootCAs = certPool
	}

	if config.Client.Tls.Certificate != "" || config.Client.Tls.Key != "" {
		if config.Client.Tls.Certificate == "" || config.Client.Tls.Key == "" {
			panic(errors.New("provide either both tls.key and tls.certificate or neither"))
		}

		tlsClientCertX509, err := tls.LoadX509KeyPair(config.Client.Tls.Certificate, config.Client.Tls.Key)
		if err != nil {
			panic(err)
		}
		tlsConfig.Certificates = []tls.Certificate{tlsClientCertX509}
	}

	client := clients.NewClient()
	defer client.Close()
	client.RegisterDefaultConnectors()

	cli.RegisterShutdownSignals(func() {
		client.Close()
		os.Exit(0)
	})

	if config.Client.Proxy != "" {
		proxyUrl, err := url.Parse(config.Client.Proxy)
		if err != nil {
			panic(err)
		}
		client.ProxyUrl = proxyUrl
	}

	client.SocketConfigurator = &cli.PingFlagsSocketConfigurator{
		Config: &config.Tunnel.Ping,
	}
	client.SetDefaultGateway = config.Tunnel.SetDefaultGateway
	client.ServerUrl = dest
	client.SetBasicAuthFromUserInfo(userInfo)
	client.TLSConfig = tlsConfig
	client.InterfaceConfig = &config.Interface
	client.AutoReconnectDelay = config.Client.AutoReconnectDelay
	client.LoadEventConfig(&config.Scripts)

	client.ServeLoop()
}