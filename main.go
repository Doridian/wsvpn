package main

import (
	"time"
	"sync"
	"net"
	"net/http"
	"log"
	"github.com/songgao/water"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  2048,
	WriteBufferSize: 2048,
	CheckOrigin: func(r *http.Request) bool { return true },
}

var slotMutex sync.Mutex
var usedSlots map[uint64]bool = make(map[uint64]bool)

func main() {
	http.HandleFunc("/", serveWs)
	err := http.ListenAndServe("127.0.0.1:9000", nil)
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

	var slot uint64 = 2
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

	defer func() {
		slotMutex.Lock()
		delete(usedSlots, slot)
		slotMutex.Unlock()
		iface.Close()
		writeLock.Lock()
		conn.Close()
		writeLock.Unlock()
	}()

	ipServer := net.IPv4(192, 168, 3, 0).String()
	//slotB := slot + 1
	ipClient := net.IPv4(192, 168, 3, byte(slot & 0xFF)).String()

	err = configIface(iface.Name(), ipClient, ipServer)
	if err != nil {
		log.Println(err)
		return
	}

	keepAlive(conn, &writeLock)

	writeLock.Lock()
	tw, err := conn.NextWriter(websocket.TextMessage)
	if err != nil {
		return
	}

	tw.Write([]byte(ipClient))
	tw.Write([]byte{ '/', '2', '4', '|', '1', '2', '8', '0' })
	err = tw.Close()
	writeLock.Unlock()
	if err != nil {
		return
	}

	go func() {
		for {
			n, err := iface.Read(packet)
			if err != nil {
				log.Println(err)
				conn.Close()
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
}

func keepAlive(c *websocket.Conn, l *sync.Mutex) {
	timeout := time.Duration(30) * time.Second

	lastResponse := time.Now()
	c.SetPongHandler(func(msg string) error {
		lastResponse = time.Now()
		return nil
	})

	go func() {
		for {
			l.Lock()
			err := c.WriteMessage(websocket.PingMessage, []byte("keepalive"))
			l.Unlock()
			if err != nil {
				return
			}
			time.Sleep(timeout/2)
			if(time.Now().Sub(lastResponse) > timeout) {
				l.Lock()
				c.Close()
				l.Unlock()
				return
			}
		}
	}()
}
