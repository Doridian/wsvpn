package main

import (
	"github.com/gorilla/websocket"
	"github.com/songgao/water"
	"github.com/apparentlymart/go-cidr/cidr"
	"log"
	"net"
	"os"
	"net/http"
	"sync"
	"fmt"
	"time"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  2048,
	WriteBufferSize: 2048,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

var slotMutex sync.Mutex
var usedSlots map[int]bool = make(map[int]bool)

const mtu = "1280"
var subnet *net.IPNet
var ipServer net.IP
var subnetSize string

func main() {
	var err error
	_, subnet, err = net.ParseCIDR(os.Args[1])
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
	err = http.ListenAndServe("127.0.0.1:9000", nil)
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

	packet := make([]byte, 2000)

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

	err = configIface(iface, mtu, ipClient, ipServer)
	if err != nil {
		log.Println(err)
		return
	}

	keepAlive(conn, &writeLock, &wg)

	writeLock.Lock()
	tw, err := conn.NextWriter(websocket.TextMessage)
	if err != nil {
		return
	}

	tw.Write([]byte(ipClient.String()))
	tw.Write([]byte{'/'})
	tw.Write([]byte(subnetSize))
	tw.Write([]byte{'|'})
	tw.Write([]byte(mtu))
	err = tw.Close()
	writeLock.Unlock()
	if err != nil {
		return
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer conn.Close()

		for {
			n, err := iface.Read(packet)
			if err != nil {
				log.Println(err)
				break
			}
			writeLock.Lock()
			err = conn.WriteMessage(websocket.BinaryMessage, packet[:n])
			writeLock.Unlock()
			if err != nil {
				break
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer conn.Close()

		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
					log.Println(err)
				}
				break
			}
			iface.Write(msg)
		}
	}()

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
