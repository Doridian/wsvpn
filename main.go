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
	"time"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  2048,
	WriteBufferSize: 2048,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

var slotMutex sync.Mutex
var usedSlots map[int]bool = make(map[int]bool)

var mtu = flag.Int("mtu", 1280, "MTU for the tunnel")
var subnetStr = flag.String("subnet", "192.168.3.0/24", "Subnet for the tunnel clients")
var listenAddr = flag.String("listen", "127.0.0.1:9000", "Listen address for the WebSocket interface")

var subnet *net.IPNet
var ipServer net.IP
var subnetSize string

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

	http.HandleFunc("/", serveWs)
	log.Printf("VPN server online at %s, serving subnet %s with MTU %d", *listenAddr, *subnetStr, *mtu)
	err = http.ListenAndServe(*listenAddr, nil)
	if err != nil {
		panic(err)
	}
}

func serveWs(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	iface, err := water.New(water.Config{
		DeviceType: water.TUN,
	})
	if err != nil {
		log.Println(err)
		conn.Close()
		return
	}

	var slot int = 1
	slotMutex.Lock()
	for usedSlots[slot] {
		slot = slot + 1
		if slot > 250 {
			slotMutex.Unlock()
			conn.Close()
			return
		}
	}
	usedSlots[slot] = true
	slotMutex.Unlock()

	var writeLock sync.Mutex
	var wg sync.WaitGroup

	defer func() {
		slotMutex.Lock()
		delete(usedSlots, slot)
		slotMutex.Unlock()
		iface.Close()
		writeLock.Lock()
		conn.Close()
		writeLock.Unlock()
	}()

	ipClient, err := cidr.Host(subnet, slot)
	if err != nil {
		log.Println(err)
		return
	}

	err = configIface(iface, *mtu, ipClient, ipServer)
	if err != nil {
		log.Println(err)
		return
	}

	keepAlive(conn, &writeLock, &wg)

	wstun_shared.SendCommand(conn, &writeLock, "init",
		fmt.Sprintf("%s/%s", ipClient.String(), subnetSize), fmt.Sprintf("%d", *mtu))

	commandMap := make(map[string]wstun_shared.CommandHandler)

	wstun_shared.HandleSocket(iface, conn, &writeLock, &wg, commandMap)

	wg.Wait()
}

func keepAlive(c *websocket.Conn, l *sync.Mutex, wg *sync.WaitGroup) {
	timeout := time.Duration(30) * time.Second

	lastResponse := time.Now()
	c.SetPongHandler(func(msg string) error {
		lastResponse = time.Now()
		return nil
	})

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer c.Close()

		for {
			l.Lock()
			err := c.WriteMessage(websocket.PingMessage, []byte("keepalive"))
			l.Unlock()
			if err != nil {
				return
			}
			time.Sleep(timeout / 2)
			if time.Now().Sub(lastResponse) > timeout {
				return
			}
		}
	}()
}
