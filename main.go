package main

import (
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/songgao/water"
	"log"
	"net"
	"os"
	"strings"
)

type remoteNet struct {
	ip    net.IP
	ipNet *net.IPNet
	str   string
}

func (r *remoteNet) getClientIP() string {
	return r.ip.String()
}

func (r *remoteNet) getServerIP() string {
	return r.ipNet.IP.To4().String()
}

func (r *remoteNet) getNetmask() string {
	mask := r.ipNet.Mask
	return fmt.Sprintf("%d.%d.%d.%d", mask[0], mask[1], mask[2], mask[3])
}

func parseRemoteNet(rNetStr string) (*remoteNet, error) {
	ip, ipNet, err := net.ParseCIDR(rNetStr)
	if err != nil {
		return nil, err
	}
	return &remoteNet{
		ip:    ip,
		ipNet: ipNet,
		str:   rNetStr,
	}, nil
}

func main() {
	dest := os.Args[1]
	log.Printf("Connecting to %s", dest)

	conn, _, err := websocket.DefaultDialer.Dial(dest, nil)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	_, msg, err := conn.ReadMessage()
	if err != nil {
		panic(err)
	}
	str := strings.Split(string(msg), "|")
	rNetStr := str[0]
	mtu := str[1]

	cRemoteNet, err := parseRemoteNet(rNetStr)
	if err != nil {
		panic(err)
	}

	log.Printf("Network %s, mtu %s", cRemoteNet.str, mtu)

	ifconfig := getPlatformSpecifics(cRemoteNet, mtu, water.Config{
		DeviceType: water.TUN,
	})
	iface, err := water.New(ifconfig)
	if err != nil {
		panic(err)
	}

	log.Printf("Opened %s", iface.Name())

	err = configIface(iface, cRemoteNet, mtu, false)
	if err != nil {
		panic(err)
	}

	log.Printf("Configured interface. Starting operations.")

	packet := make([]byte, 2000)

	go func() {
		for {
			n, err := iface.Read(packet)
			if err != nil {
				log.Println(err)
				conn.Close()
				break
			}
			err = conn.WriteMessage(websocket.BinaryMessage, packet[:n])
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
