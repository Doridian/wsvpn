package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/Doridian/wsvpn/shared"
	"github.com/Doridian/wsvpn/shared/commands"
	"github.com/Doridian/wsvpn/shared/sockets"
	"github.com/Doridian/wsvpn/shared/sockets/adapters"
	"github.com/gorilla/websocket"
	"github.com/marten-seemann/webtransport-go"
	"github.com/songgao/water"
)

var defaultGateway = flag.Bool("default-gateway", false, "Route all traffic through VPN")
var connectAddr = flag.String("connect", "", "Server address to connect to (ex. ws://example.com:9000)")
var authFile = flag.String("auth-file", "", "File to read authentication from in the format user:password")

var upScript = flag.String("up-script", "", "Script to run once the VPN is online")
var downScript = flag.String("down-script", "", "Script to run when the VPN goes offline")

var proxyAddr = flag.String("proxy", "", "HTTP proxy to use for connection (ex. http://example.com:8080)")

var ifaceName = flag.String("interface-name", "", "Interface name of the interface to use")

var caCertFile = flag.String("ca-certificates", "", "If specified, use all PEM certs in this file as valid root certs only")
var insecure = flag.Bool("insecure", false, "Disable all TLS verification")
var tlsClientCert = flag.String("tls-client-cert", "", "TLS certificate file for client authentication")
var tlsClientKey = flag.String("tls-client-key", "", "TLS key file for client authentication")

func runEventScript(script *string, op string, cRemoteNet *remoteNet, iface *water.Interface) error {
	if script == nil {
		return nil
	}
	scriptStr := *script
	if scriptStr == "" {
		return nil
	}

	return shared.ExecCmd(scriptStr, op, cRemoteNet.str, iface.Name())
}

func main() {
	flag.Usage = shared.UsageWithVersion
	flag.Parse()

	destUrlString := *connectAddr
	if destUrlString == "" {
		flag.Usage()
		return
	}

	shared.PrintVersion("C")

	dest, err := url.Parse(destUrlString)
	if err != nil {
		panic(err)
	}
	dest.Scheme = strings.ToLower(dest.Scheme)

	authFileString := *authFile
	var userInfo *url.Userinfo

	if authFileString != "" {
		authData, err := ioutil.ReadFile(authFileString)
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

	if dest.User != nil {
		dest.User = nil
		log.Printf("[C] WARNING: You have put your password on the command line! This can cause security issues!")
	}

	tlsConfig := &tls.Config{}

	tlsConfig.InsecureSkipVerify = *insecure
	shared.TlsUseFlags(tlsConfig)

	if tlsConfig.InsecureSkipVerify {
		log.Printf("[C] WARNING: TLS verification disabled! This can cause security issues!")
	}

	caCertFileString := *caCertFile
	if caCertFileString != "" {
		data, err := ioutil.ReadFile(caCertFileString)
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

	tlsClientCertStr := *tlsClientCert
	tlsClientKeyStr := *tlsClientKey
	if tlsClientCertStr != "" || tlsClientKeyStr != "" {
		if tlsClientCertStr == "" || tlsClientKeyStr == "" {
			panic(errors.New("provide either both tls-client-key and tls-client-cert or neither"))
		}

		tlsClientCertX509, err := tls.LoadX509KeyPair(tlsClientCertStr, tlsClientKeyStr)
		if err != nil {
			panic(err)
		}
		tlsConfig.Certificates = []tls.Certificate{tlsClientCertX509}
	}

	header := http.Header{}
	if userInfo != nil {
		if tlsClientCertStr == "" {
			log.Printf("[C] Connecting to %s as user %s", dest.Redacted(), userInfo.Username())
		} else {
			log.Printf("[C] Connecting to %s as user %s with mutual TLS authentication", dest.Redacted(), userInfo.Username())
		}
		if _, pws := userInfo.Password(); !pws {
			log.Printf("[C] WARNING: You have specified to connect with a username but without a password!")
		}
		header.Add("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(userInfo.String())))
	} else if tlsClientCertStr == "" {
		log.Printf("[C] WARNING: Connecting to %s without authentication!", dest.Redacted())
	} else {
		log.Printf("[C] Connecting to %s with mutual TLS authentication", dest.Redacted())
	}

	var adapter adapters.SocketAdapter
	switch dest.Scheme {
	case "webtransport":
		dest.Scheme = "https"
		dialer := webtransport.Dialer{}
		dialer.TLSClientConf = tlsConfig

		if *proxyAddr != "" {
			panic(errors.New("proxy is not support for WebTransport at the moment"))
		}

		_, conn, err := dialer.Dial(context.Background(), dest.String(), header)
		if err != nil {
			panic(err)
		}

		adapter = adapters.NewWebTransportAdapter(conn, false)
	case "ws":
	case "wss":
		dialer := websocket.Dialer{}
		proxyUrlString := *proxyAddr
		if proxyUrlString != "" {
			proxyUrl, err := url.Parse(proxyUrlString)
			if err != nil {
				panic(err)
			}
			log.Printf("[C] Using HTTP proxy %s", proxyUrl.Redacted())
			dialer.Proxy = func(_ *http.Request) (*url.URL, error) {
				return proxyUrl, nil
			}
		}
		dialer.TLSClientConfig = tlsConfig

		conn, _, err := dialer.Dial(dest.String(), header)
		if err != nil {
			panic(err)
		}

		adapter = adapters.NewWebSocketAdapter(conn)
	default:
		panic(fmt.Errorf("invalid protocol: %s", dest.Scheme))
	}

	defer adapter.Close()
	connId := "0"

	tlsConnState, ok := adapter.GetTLSConnectionState()
	if ok {
		log.Printf("[%s] TLS %s %s connection established with cipher=%s", connId, shared.TlsVersionString(tlsConnState.Version), adapter.Name(), tls.CipherSuiteName(tlsConnState.CipherSuite))
	} else {
		log.Printf("[%s] Unencrypted %s connection established", connId, adapter.Name())
	}

	var iface *water.Interface
	var cRemoteNet *remoteNet

	defer func() {
		if iface != nil {
			if cRemoteNet != nil {
				runEventScript(downScript, "down", cRemoteNet, iface)
			}
			iface.Close()
		}
	}()

	socket := sockets.MakeSocket(connId, adapter, nil, false)
	socket.AddCommandHandler(commands.AddRouteCommandName, func(command *commands.IncomingCommand) error {
		var parameters commands.AddRouteParameters
		err := json.Unmarshal(command.Parameters, &parameters)
		if err != nil {
			return err
		}

		if iface == nil || cRemoteNet == nil {
			return errors.New("cannot addroute before init")
		}

		_, routeNet, err := net.ParseCIDR(parameters.Route)
		if err != nil {
			return err
		}

		return addRoute(iface, cRemoteNet, routeNet)
	})

	socket.AddCommandHandler(commands.InitCommandName, func(command *commands.IncomingCommand) error {
		var err error
		var parameters commands.InitParameters

		err = json.Unmarshal(command.Parameters, &parameters)
		if err != nil {
			return err
		}

		cRemoteNet, err = parseRemoteNet(parameters.IpAddress)
		if err != nil {
			return err
		}

		log.Printf("[%s] Network mode %s, subnet %s, mtu %d", connId, parameters.Mode, cRemoteNet.str, parameters.MTU)

		socket.SetMTU(parameters.MTU)

		var waterMode water.DeviceType
		if parameters.Mode == "TUN" {
			waterMode = water.TUN
		} else {
			waterMode = water.TAP
		}

		ifconfig := getPlatformSpecifics(cRemoteNet, parameters.MTU, *ifaceName, water.Config{
			DeviceType: waterMode,
		})
		iface, err = water.New(ifconfig)
		if err != nil {
			return err
		}

		log.Printf("[%s] Opened %s", connId, iface.Name())

		err = configIface(iface, parameters.Mode != "TAP_NOCONF", cRemoteNet, parameters.MTU, *defaultGateway)
		if err != nil {
			return err
		}

		log.Printf("[%s] Configured interface, starting operations", connId)
		err = socket.SetInterface(iface)
		if err != nil {
			return err
		}

		go runEventScript(upScript, "up", cRemoteNet, iface)

		return nil
	})
	socket.Serve()
	socket.Wait()
}
