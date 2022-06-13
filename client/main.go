package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"flag"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/Doridian/wsvpn/shared"
	"github.com/gorilla/websocket"
	"github.com/songgao/water"
)

const DEFAULT_URL = "ws://example.com"

var defaultGateway = flag.Bool("default-gateway", false, "Route all traffic through VPN")
var connectAddr = flag.String("connect", DEFAULT_URL, "Server address to connect to")
var authFile = flag.String("auth-file", "", "File to read authentication from in the format user:password")
var upScript = flag.String("up-script", "", "Script to run once the VPN is online")
var downScript = flag.String("down-script", "", "Script to run when the VPN goes offline")
var proxyAddr = flag.String("proxy", "", "HTTP proxy to use for connection (ex. http://10.10.10.10:8080)")

var caCertFile = flag.String("ca-certificates", "", "If specified, use all PEM certs in this file as valid root certs only")
var insecure = flag.Bool("insecure", false, "Disable all TLS verification")

func productionWarnings(str string) {
	for n := 0; n <= 5; n++ {
		log.Printf("DO NOT USE THIS IN PRODUCTION! %s!", str)
	}
}

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
	flag.Parse()

	destUrlString := *connectAddr
	if destUrlString == DEFAULT_URL {
		flag.PrintDefaults()
		return
	}

	dest, err := url.Parse(destUrlString)
	if err != nil {
		panic(err)
	}

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
		productionWarnings("PASSWORD ON THE COMMAND LINE")
	}

	header := http.Header{}
	if userInfo != nil {
		log.Printf("Connecting to %s as user %s", dest.Redacted(), userInfo.Username())
		if _, pws := userInfo.Password(); !pws {
			productionWarnings("NO PASSWORD SET")
		}
		header.Add("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(userInfo.String())))
	} else {
		log.Printf("Connecting to %s without authentication", dest.String())
		productionWarnings("NO AUTHENTICATION SET")
	}

	dialer := websocket.Dialer{}

	proxyUrlString := *proxyAddr
	if proxyUrlString != "" {
		proxyUrl, err := url.Parse(proxyUrlString)
		if err != nil {
			panic(err)
		}
		log.Printf("Using HTTP proxy %s", proxyUrl.Redacted())
		dialer.Proxy = func(_ *http.Request) (*url.URL, error) {
			return proxyUrl, nil
		}
	}

	tlsConfig := &tls.Config{}

	dialer.TLSClientConfig = tlsConfig
	tlsConfig.InsecureSkipVerify = *insecure
	shared.TlsUseFlags(tlsConfig)

	if tlsConfig.InsecureSkipVerify {
		productionWarnings("TLS VERIFICATION DISABLED")
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

	conn, _, err := dialer.Dial(dest.String(), header)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	websocketTlsConn, ok := conn.UnderlyingConn().(*tls.Conn)
	if ok {
		connState := websocketTlsConn.ConnectionState()
		log.Printf("TLS %s connection established with cipher=%s", shared.TlsVersionString(connState.Version), tls.CipherSuiteName(connState.CipherSuite))
	} else {
		log.Printf("Unencrypted connection established")
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

	socket := shared.MakeSocket("0", conn, nil, false)
	socket.AddCommandHandler("addroute", func(args []string) error {
		if iface == nil || cRemoteNet == nil {
			return errors.New("cannot addroute before init")
		}

		if len(args) < 1 {
			return errors.New("addroute needs 1 argument")
		}
		_, routeNet, err := net.ParseCIDR(args[0])
		if err != nil {
			return err
		}
		return addRoute(iface, cRemoteNet, routeNet)
	})
	socket.AddCommandHandler("init", func(args []string) error {
		var err error

		mode := args[0]

		rNetStr := args[1]
		mtu, err := strconv.Atoi(args[2])
		if err != nil {
			panic(err)
		}

		cRemoteNet, err = parseRemoteNet(rNetStr)
		if err != nil {
			panic(err)
		}

		log.Printf("Network mode %s, subnet %s, mtu %d", mode, cRemoteNet.str, mtu)

		var waterMode water.DeviceType
		if mode == "TUN" {
			waterMode = water.TUN
		} else {
			waterMode = water.TAP
		}

		ifconfig := getPlatformSpecifics(cRemoteNet, mtu, water.Config{
			DeviceType: waterMode,
		})
		iface, err = water.New(ifconfig)
		if err != nil {
			panic(err)
		}

		log.Printf("Opened %s", iface.Name())

		err = configIface(iface, mode != "TAP_NOCONF", cRemoteNet, mtu, *defaultGateway)
		if err != nil {
			panic(err)
		}

		log.Printf("Configured interface, starting operations")
		err = socket.SetInterface(iface)
		if err != nil {
			panic(err)
		}

		go runEventScript(upScript, "up", cRemoteNet, iface)

		return nil
	})
	socket.Serve()
	socket.Wait()
}
