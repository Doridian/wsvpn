package main

import (
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"

	"github.com/Doridian/wsvpn/server/authenticators"
	"github.com/Doridian/wsvpn/shared"
	"github.com/apparentlymart/go-cidr/cidr"
	"github.com/gorilla/websocket"
	"github.com/songgao/water"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  2048,
	WriteBufferSize: 2048,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

var slotMutex sync.Mutex
var ifaceCreationMutex sync.Mutex
var usedSlots map[uint64]bool = make(map[uint64]bool)

var mtu = flag.Int("mtu", 1280, "MTU for the tunnel")
var subnetStr = flag.String("subnet", "192.168.3.0/24", "Subnet for the tunnel clients")
var listenAddr = flag.String("listen", "127.0.0.1:9000", "Listen address for the WebSocket interface")

var authenticatorStrPtr = flag.String("authenticator", "allow-all", "Which authenticator to use (allow-all, htpasswd)")
var authenticatorConfigStrPtr = flag.String("authenticator-config", "", "Authenticator config file (ex. htpasswd file for htpasswd authenticator, empty for default)")

var useTap = flag.Bool("tap", false, "Use a TAP and not a TUN")
var useTapNoConf = flag.Bool("tap-remote-noconf", false, "Do not send IP or route configuration with TAP to remote")
var useTapIfaceNoConf = flag.Bool("tap-local-noconf", false, "Do not configure local TAP interface at all except MTU")
var useClientToClient = flag.Bool("allow-client-to-client", false, "Allow client-to-client communication (in TAP)")

var tlsCert = flag.String("tls-cert", "", "TLS certificate file for listener")
var tlsKey = flag.String("tls-key", "", "TLS key file for listener")

var subnet *net.IPNet
var ipServer net.IP
var subnetSize string
var maxSlot uint64

var tapMode bool
var tapDev *water.Interface
var modeString string

var authenticator authenticators.Authenticator

func main() {
	flag.Usage = shared.UsageWithVersion
	flag.Parse()

	shared.PrintVersion()

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

		shared.SetMultiClientIfaceMode(true)
	} else {
		shared.SetMultiClientIfaceMode(false)
		modeString = "TUN"
	}

	shared.SetClientToClient(*useClientToClient)

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

	log.Printf("[S] VPN server online at %s, mode %s, serving subnet %s (%d max clients) with MTU %d",
		*listenAddr, modeString, *subnetStr, maxSlot-1, *mtu)

	http.HandleFunc("/", serveWs)

	tlsCertStr := *tlsCert
	tlsKeyStr := *tlsKey
	if tlsCertStr != "" || tlsKeyStr != "" {
		if tlsCertStr == "" || tlsKeyStr == "" {
			panic(errors.New("provide either both tls-key and tls-cert or neither"))
		}

		tlsConfig := &tls.Config{}
		shared.TlsUseFlags(tlsConfig)

		server := http.Server{
			Addr:      *listenAddr,
			TLSConfig: tlsConfig,
		}

		err = server.ListenAndServeTLS(tlsCertStr, tlsKeyStr)
	} else {
		err = http.ListenAndServe(*listenAddr, nil)
	}
	if err != nil {
		panic(err)
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

		var s *shared.Socket
		if isUnicast {
			s = shared.FindSocketByMAC(dest)
			if s != nil {
				s.WriteMessage(websocket.BinaryMessage, packet[:n])
			}
		} else {
			shared.BroadcastMessage(websocket.BinaryMessage, packet[:n], nil)
		}
	}
}

func serveWs(w http.ResponseWriter, r *http.Request) {
	authResult := authenticator.Authenticate(r, w)
	if authResult != authenticators.AUTH_OK {
		if authResult == authenticators.AUTH_FAILED_DEFAULT {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
		}
		return
	}

	var err error

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[S] Error upgrading to WS: %v", err)
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
			log.Printf("[S] Error extending TUN config: %v", err)
			conn.Close()
			return
		}

		iface, err = water.New(tunConfig)
		ifaceCreationMutex.Unlock()
		if err != nil {
			log.Printf("[S] Error creating new TUN: %v", err)
			conn.Close()
			return
		}
	}

	var slot uint64 = 1
	slotMutex.Lock()
	for usedSlots[slot] {
		slot = slot + 1
		if slot > maxSlot {
			slotMutex.Unlock()
			conn.Close()
			log.Println("[S] Cannot connect new client: IP slots exhausted")
			if !tapMode {
				iface.Close()
			}
			return
		}
	}
	usedSlots[slot] = true
	slotMutex.Unlock()

	connId := fmt.Sprintf("%d", slot)

	log.Printf("[%s] Client ENTER (interface %s)", connId, iface.Name())

	socket := shared.MakeSocket(connId, conn, iface, tapMode)

	defer func() {
		slotMutex.Lock()
		delete(usedSlots, slot)
		slotMutex.Unlock()
		socket.Close()
		log.Printf("[%s] Client EXIT (interface %s)", connId, iface.Name())
	}()

	ipClient, err := cidr.Host(subnet, int(slot)+1)
	if err != nil {
		log.Printf("[%s] Error transforming client IP: %v", connId, err)
		return
	}

	if !tapMode {
		err = configIface(iface, true, *mtu, ipClient, ipServer, subnet)
		if err != nil {
			log.Printf("[%s] Error configuring interface: %v", connId, err)
			return
		}
	}

	socket.SendCommand("init", modeString, fmt.Sprintf("%s/%s", ipClient.String(), subnetSize), fmt.Sprintf("%d", *mtu))
	socket.Serve()
	socket.Wait()
}
