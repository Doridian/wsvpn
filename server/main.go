package main

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"sync"

	"github.com/Doridian/wsvpn/server/authenticators"
	"github.com/Doridian/wsvpn/shared"
	"github.com/Doridian/wsvpn/shared/commands"
	"github.com/Doridian/wsvpn/shared/sockets"
	"github.com/Doridian/wsvpn/shared/sockets/adapters"
	"github.com/apparentlymart/go-cidr/cidr"
	"github.com/gorilla/websocket"
	"github.com/lucas-clemente/quic-go/http3"
	"github.com/marten-seemann/webtransport-go"
	"github.com/songgao/water"
)

var webSocketUpgrader = &websocket.Upgrader{
	ReadBufferSize:  2048,
	WriteBufferSize: 2048,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

var webTransportServer *webtransport.Server

var slotMutex sync.Mutex
var ifaceCreationMutex sync.Mutex
var usedSlots map[uint64]bool = make(map[uint64]bool)

var mtu = flag.Int("mtu", 1280, "MTU for the tunnel")
var subnetStr = flag.String("subnet", "192.168.3.0/24", "Subnet for the tunnel clients")
var listenAddr = flag.String("listen", "127.0.0.1:9000", "Listen address for the WebSocket interface")
var listenHTTP3Enable = flag.Bool("enable-http3", false, "Enable HTTP/3 protocol")

var authenticatorStrPtr = flag.String("authenticator", "allow-all", "Which authenticator to use (allow-all, htpasswd)")
var authenticatorConfigStrPtr = flag.String("authenticator-config", "", "Authenticator config file (ex. htpasswd file for htpasswd authenticator, empty for default)")

var useTap = flag.Bool("tap", false, "Use a TAP and not a TUN")
var useTapNoConf = flag.Bool("tap-remote-noconf", false, "Do not send IP or route configuration with TAP to remote")
var useTapIfaceNoConf = flag.Bool("tap-local-noconf", false, "Do not configure local TAP interface at all except MTU")
var useClientToClient = flag.Bool("allow-client-to-client", false, "Allow client-to-client communication (in TAP)")

var tlsCert = flag.String("tls-cert", "", "TLS certificate file for listener")
var tlsKey = flag.String("tls-key", "", "TLS key file for listener")
var tlsClientCA = flag.String("tls-client-ca", "", "If set, performs TLS client certificate authentication based on given CA certificate")

var subnet *net.IPNet
var ipServer net.IP
var subnetSize string
var maxSlot uint64

var tapMode bool
var tapDev *water.Interface
var modeString string
var http3Enabled bool

var authenticator authenticators.Authenticator

func main() {
	flag.Usage = shared.UsageWithVersion
	flag.Parse()

	shared.PrintVersion("S")

	var err error
	_, subnet, err = net.ParseCIDR(*subnetStr)
	if err != nil {
		panic(err)
	}
	ipServer, err = cidr.Host(subnet, 1)
	if err != nil {
		panic(err)
	}
	subnetOnes, _ := subnet.Mask.Size()
	subnetSize = fmt.Sprintf("%d", subnetOnes)

	maxSlot = cidr.AddressCount(subnet) - 2

	tapMode = *useTap

	err = verifyPlatformFlags()
	if err != nil {
		panic(err)
	}

	if tapMode {
		ifaceCreationMutex.Lock()
		tapConfig := water.Config{
			DeviceType: water.TAP,
		}
		err = extendTAPConfig(&tapConfig)
		if err != nil {
			panic(err)
		}

		tapDev, err = water.New(tapConfig)
		if err != nil {
			panic(err)
		}
		ifaceCreationMutex.Unlock()

		if *useTapNoConf {
			modeString = "TAP_NOCONF"
		} else {
			modeString = "TAP"
		}

		err = configIface(tapDev, !*useTapIfaceNoConf, *mtu, ipServer, ipServer, subnet)
		if err != nil {
			panic(err)
		}

		sockets.SetMultiClientIfaceMode(true)
	} else {
		sockets.SetMultiClientIfaceMode(false)
		modeString = "TUN"
	}

	sockets.SetClientToClient(*useClientToClient)

	authenticatorStr := *authenticatorStrPtr
	if authenticatorStr == "allow-all" {
		authenticator = &authenticators.AllowAllAuthenticator{}
	} else if authenticatorStr == "htpasswd" {
		authenticator = &authenticators.HtpasswdAuthenticator{}
	} else {
		panic(errors.New("invalid authenticator selected"))
	}

	err = authenticator.Load(*authenticatorConfigStrPtr)
	if err != nil {
		panic(err)
	}

	if tapMode {
		go serveTap()
	}

	httpHandlerFunc := http.HandlerFunc(serveSocket)
	http3Enabled = *listenHTTP3Enable

	log.Printf("[S] VPN server online at %s (HTTP/3 %s), mode %s, serving subnet %s (%d max clients) with MTU %d",
		*listenAddr, shared.BoolToString(http3Enabled, "enabled", "disabled"), modeString, *subnetStr, maxSlot-1, *mtu)

	tlsCertStr := *tlsCert
	tlsKeyStr := *tlsKey
	tlsClientCAStr := *tlsClientCA
	if tlsCertStr != "" || tlsKeyStr != "" || tlsClientCAStr != "" {
		if tlsCertStr == "" && tlsKeyStr == "" {
			panic(errors.New("tls-client-ca requires tls-key and tls-cert"))
		}

		if tlsCertStr == "" || tlsKeyStr == "" {
			panic(errors.New("provide either both tls-key and tls-cert or neither"))
		}

		tlsConfig := &tls.Config{}

		if tlsClientCAStr != "" {
			tlsClientCAPEM, err := ioutil.ReadFile(tlsClientCAStr)
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

		shared.TlsUseFlags(tlsConfig)

		http3Wait := &sync.WaitGroup{}

		if http3Enabled {
			http3Wait.Add(1)

			quicServer := http3.Server{
				Addr:      *listenAddr,
				TLSConfig: tlsConfig,
				Handler:   httpHandlerFunc,
			}

			webTransportServer = &webtransport.Server{
				H3:          quicServer,
				CheckOrigin: func(r *http.Request) bool { return true },
			}

			go func() {
				defer http3Wait.Done()
				err := webTransportServer.ListenAndServeTLS(tlsCertStr, tlsKeyStr)
				if err != nil {
					panic(err)
				}
			}()

			httpHandlerFunc = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				quicServer.SetQuicHeaders(w.Header())
				serveSocket(w, r)
			})
		}

		server := http.Server{
			Addr:      *listenAddr,
			TLSConfig: tlsConfig,
			Handler:   httpHandlerFunc,
		}

		err = server.ListenAndServeTLS(tlsCertStr, tlsKeyStr)
		if err != nil {
			panic(err)
		}

		http3Wait.Wait()
	} else {
		if http3Enabled {
			panic(errors.New("HTTP/3 requires TLS"))
		}
		server := http.Server{
			Addr:    *listenAddr,
			Handler: httpHandlerFunc,
		}
		err = server.ListenAndServe()
		if err != nil {
			panic(err)
		}
	}
}

func serveTap() {
	defer panic(errors.New("TAP closed"))

	packet := make([]byte, 2000)

	for {
		n, err := tapDev.Read(packet)
		if err != nil {
			log.Printf("[S] Error reading packet from tap: %v", err)
			return
		}
		// Ignore everything that isn't an ethernet frame
		if n < 14 {
			continue
		}
		dest := shared.GetDestMAC(packet)
		isUnicast := shared.MACIsUnicast(dest)

		var s *sockets.Socket
		if isUnicast {
			s = sockets.FindSocketByMAC(dest)
			if s != nil {
				s.WriteDataMessage(packet[:n])
			}
		} else {
			sockets.BroadcastDataMessage(packet[:n], nil)
		}
	}
}

func handleAuth(w http.ResponseWriter, r *http.Request) (bool, string) {
	tlsUsername := ""
	if r.TLS != nil && len(r.TLS.PeerCertificates) > 0 {
		tlsUsername = r.TLS.PeerCertificates[0].Subject.CommonName
	}
	authResult, authUsername := authenticator.Authenticate(r, w)
	if authResult != authenticators.AUTH_OK {
		if authResult == authenticators.AUTH_FAILED_DEFAULT {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
		}
		return false, ""
	}

	if authUsername != "" && tlsUsername != "" && authUsername != tlsUsername {
		http.Error(w, "Mutual TLS CN is not equal authenticator username", http.StatusUnauthorized)
		return false, ""
	}

	if authUsername == "" {
		authUsername = tlsUsername
	}

	return true, authUsername
}

func serveWebSocket(w http.ResponseWriter, r *http.Request) (adapters.SocketAdapter, error) {
	conn, err := webSocketUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, err
	}
	return adapters.NewWebSocketAdapter(conn), nil
}

func serveWebTransport(w http.ResponseWriter, r *http.Request) (adapters.SocketAdapter, error) {
	conn, err := webTransportServer.Upgrade(w, r)
	if err != nil {
		return nil, err
	}
	return adapters.NewWebTransportAdapter(conn, true), nil
}

func serveSocket(w http.ResponseWriter, r *http.Request) {
	authOk, authUsername := handleAuth(w, r)
	if !authOk {
		return
	}

	var slot uint64 = 1
	slotMutex.Lock()
	for usedSlots[slot] {
		slot = slot + 1
		if slot > maxSlot {
			slotMutex.Unlock()
			log.Println("[S] Cannot connect new client: IP slots exhausted")
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	}
	usedSlots[slot] = true
	slotMutex.Unlock()

	defer func() {
		slotMutex.Lock()
		delete(usedSlots, slot)
		slotMutex.Unlock()
	}()

	connId := fmt.Sprintf("%d", slot)

	var err error
	var adapter adapters.SocketAdapter
	if r.Proto == "webtransport" && http3Enabled {
		adapter, err = serveWebTransport(w, r)
	} else {
		adapter, err = serveWebSocket(w, r)
	}

	if err != nil {
		log.Printf("[%s] Error upgrading connection: %v", connId, err)
		return
	}

	defer adapter.Close()

	tlsConnState, ok := adapter.GetTLSConnectionState()
	if ok {
		log.Printf("[%s] TLS %s %s connection established with cipher=%s", connId, shared.TlsVersionString(tlsConnState.Version), adapter.Name(), tls.CipherSuiteName(tlsConnState.CipherSuite))
	} else {
		log.Printf("[%s] Unencrypted %s connection established", connId, adapter.Name())
	}

	if authUsername != "" {
		log.Printf("[%s] Client authenticated as %s", connId, authUsername)
	}

	ipClient, err := cidr.Host(subnet, int(slot)+1)
	if err != nil {
		log.Printf("[%s] Error transforming client IP: %v", connId, err)
		return
	}

	var iface *water.Interface
	if tapMode {
		iface = tapDev
	} else {
		ifaceCreationMutex.Lock()
		tunConfig := water.Config{
			DeviceType: water.TUN,
		}
		err = extendTUNConfig(&tunConfig)
		if err != nil {
			ifaceCreationMutex.Unlock()
			log.Printf("[%s] Error extending TUN config: %v", connId, err)
			return
		}

		iface, err = water.New(tunConfig)
		ifaceCreationMutex.Unlock()
		if err != nil {
			log.Printf("[%s] Error creating new TUN: %v", connId, err)
			return
		}

		defer iface.Close()

		log.Printf("[%s] Assigned interface %s", connId, iface.Name())

		err = configIface(iface, true, *mtu, ipClient, ipServer, subnet)
		if err != nil {
			log.Printf("[%s] Error configuring interface: %v", connId, err)
			return
		}
	}

	socket := sockets.MakeSocket(connId, adapter, iface, tapMode)
	socket.SetMTU(*mtu)
	defer socket.Close()

	log.Printf("[%s] Connection fully established", connId)
	defer log.Printf("[%s] Disconnected", connId)

	socket.Serve()
	socket.MakeAndSendCommand(&commands.InitParameters{Mode: modeString, IpAddress: fmt.Sprintf("%s/%s", ipClient.String(), subnetSize), MTU: *mtu})
	socket.Wait()
}
