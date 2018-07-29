package main

import (
	"flag"
	"fmt"
	"github.com/Doridian/wstun_shared"
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

var subnet *net.IPNet
var ipServer net.IP
var subnetSize string
var maxSlot uint64

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

	log.Printf("VPN server online at %s, serving subnet %s (%d max clients) with MTU %d",
		*listenAddr, *subnetStr, maxSlot-1, *mtu)

	http.HandleFunc("/", serveWs)
	err = http.ListenAndServe(*listenAddr, nil)
	if err != nil {
		panic(err)
	}
}

func serveWs(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Error upgrading to WS: %v", err)
		return
	}

	iface, err := water.New(water.Config{
		DeviceType: water.TUN,
	})
	if err != nil {
		log.Printf("Error creating new TUN: %v", err)
		conn.Close()
		return
	}

	var slot uint64 = 1
	slotMutex.Lock()
	for usedSlots[slot] {
		slot = slot + 1
		if slot > maxSlot {
			slotMutex.Unlock()
			conn.Close()
			log.Println("Cannot connect new client. IP slots exhausted.")
			return
		}
	}
	usedSlots[slot] = true
	slotMutex.Unlock()

	connId := fmt.Sprintf("%d", slot)

	log.Printf("[%s] Client ENTER", connId)

	socket := wstun_shared.MakeSocket(connId, conn, iface)

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

	err = configIface(iface, *mtu, ipClient, ipServer)
	if err != nil {
		log.Printf("[%s] Error configuring interface: %v", connId, err)
		return
	}

	socket.SendCommand("init", fmt.Sprintf("%s/%s", ipClient.String(), subnetSize), fmt.Sprintf("%d", *mtu))
	socket.Serve()
	socket.Wait()
}
