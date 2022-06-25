package main

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"flag"
	"io/ioutil"
	"log"
	"net/url"
	"strings"

	"github.com/Doridian/wsvpn/client/clients"
	"github.com/Doridian/wsvpn/shared"
)

var defaultGateway = flag.Bool("default-gateway", false, "Route all traffic through VPN")
var connectAddr = flag.String("connect", "", "Server address to connect to (ex. ws://example.com:9000)")
var authFile = flag.String("auth-file", "", "File to read authentication from in the format user:password")

var proxyAddr = flag.String("proxy", "", "HTTP proxy to use for connection (ex. http://example.com:8080)")

var ifaceName = flag.String("interface-name", "", "Interface name of the interface to use")

var caCertFile = flag.String("ca-certificates", "", "If specified, use all PEM certs in this file as valid root certs only")
var insecure = flag.Bool("insecure", false, "Disable all TLS verification")
var tlsClientCert = flag.String("tls-client-cert", "", "TLS certificate file for client authentication")
var tlsClientKey = flag.String("tls-client-key", "", "TLS key file for client authentication")

var upScript = flag.String("up-script", "", "Script to run once the VPN is online")
var downScript = flag.String("down-script", "", "Script to run when the VPN goes offline")

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

	client := clients.NewClient()
	defer client.Close()

	proxyAddrString := *proxyAddr
	if proxyAddrString != "" {
		proxyUrl, err := url.Parse(proxyAddrString)
		if err != nil {
			panic(err)
		}
		client.ProxyUrl = proxyUrl
	}

	client.SetDefaultGateway = *defaultGateway
	client.ServerUrl = dest
	client.InterfaceName = *ifaceName
	client.SetBasicAuthFromUserInfo(userInfo)
	client.TLSConfig = tlsConfig

	client.Serve()
	client.Wait()
}
