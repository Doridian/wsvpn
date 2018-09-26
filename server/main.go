package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/Doridian/wsvpn/shared"
	"github.com/apparentlymart/go-cidr/cidr"
	"github.com/gorilla/websocket"
	"github.com/songgao/water"
	"log"
	"net"
	"net/http"
	"sync"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  2048,
	WriteBufferSize: 2048,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

var slotMutex sync.Mutex
var usedSlots map[uint64]bool = make(map[uint64]bool)

var mtu = flag.Int("mtu", 1280, "MTU for the tunnel")
var subnetStr = flag.String("subnet", "192.168.3.0/24", "Subnet for the tunnel clients")
var listenAddr = flag.String("listen", "127.0.0.1:9000", "Listen address for the WebSocket interface")
var useTap = flag.Bool("tap", false, "Use a TAP and not a TUN")
var useTapNoConf = flag.Bool("tap-noconf", false, "Do not send IP config with TAP (and do not configure tap interface; ignore -subnet)")
var useUid = flag.Int("uid", 0, "setuid() after opening TAP")
var useGid = flag.Int("gid", 0, "setgid() after opening TAP")

var subnet *net.IPNet
var ipServer net.IP
var subnetSize string
var maxSlot uint64

var tapMode bool
var tapDev *water.Interface
var modeString string

func main() {
	flag.Parse()

	var err error
	_, subnet, err = net.ParseCIDR(*subnetStr)
	if err != nil {
		panic(err)
	}
	ipServer, err = cidr.Host(subnet, 0)
	if err != nil {
		panic(err)
	}
	subnetOnes, _ := subnet.Mask.Size()
	subnetSize = fmt.Sprintf("%d", subnetOnes)

	maxSlot = cidr.AddressCount(subnet)

	tapMode = *useTap
	if tapMode {
		tapDev, err = water.New(water.Config{
			DeviceType: water.TAP,
		})
		if err != nil {
			panic(err)
		}

		if *useTapNoConf {
			modeString = "TAP_NOCONF"
		} else {
			modeString = "TAP"
		}

		err = configIface(tapDev, !*useTapNoConf, *mtu, ipServer, ipServer)
		if err != nil {
			panic(err)
		}

		shared.SetMACLearning(true)

		go serveTap()
	} else {
		shared.SetMACLearning(false)
		modeString = "TUN"
	}

	uid := *useUid
	gid := *useGid
	if uid > 0 || gid > 0 {
		setProcessUidGid(uid, gid)
	}

	log.Printf("[S] VPN server online at %s, mode %s, serving subnet %s (%d max clients) with MTU %d",
		*listenAddr, modeString, *subnetStr, maxSlot-1, *mtu)

	http.HandleFunc("/", serveWs)
	err = http.ListenAndServe(*listenAddr, nil)
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
		iface, err = water.New(water.Config{
			DeviceType: water.TUN,
		})
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
			log.Println("[S] Cannot connect new client. IP slots exhausted.")
			return
		}
	}
	usedSlots[slot] = true
	slotMutex.Unlock()

	connId := fmt.Sprintf("%d", slot)

	log.Printf("[%s] Client ENTER", connId)

	socket := shared.MakeSocket(connId, conn, iface, tapMode)

	defer func() {
		slotMutex.Lock()
		delete(usedSlots, slot)
		slotMutex.Unlock()
		socket.Close()
		log.Printf("[%s] Client EXIT", connId)
	}()

	ipClient, err := cidr.Host(subnet, int(slot))
	if err != nil {
		log.Printf("[%s] Error transforming client IP: %v", connId, err)
		return
	}

	if !tapMode {
		err = configIface(iface, true, *mtu, ipClient, ipServer)
		if err != nil {
			log.Printf("[%s] Error configuring interface: %v", connId, err)
			return
		}
	}

	socket.SendCommand("init", modeString, fmt.Sprintf("%s/%s", ipClient.String(), subnetSize), fmt.Sprintf("%d", *mtu))
	socket.Serve()
	socket.Wait()
}
