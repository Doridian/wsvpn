package main

import (
	"encoding/base64"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/songgao/water"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
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
	dest, err := url.Parse(os.Args[1])
	if err != nil {
		panic(err)
	}

	userInfo := dest.User
	dest.User = nil

	header := http.Header{}
	if userInfo != nil {
		log.Printf("Connecting to %s as user %s", dest.String(), userInfo.Username())
		header.Add("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(userInfo.String())))
	} else {
		log.Printf("Connecting to %s without authentication.", dest.String())
		for n := 0; n <= 5; n++ {
			log.Printf("DO NOT USE THIS IN PRODUCTION!")
		}
	}
	conn, _, err := websocket.DefaultDialer.Dial(dest.String(), header)
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
	mtu, err := strconv.Atoi(str[1])
	if err != nil {
		panic(err)
	}

	cRemoteNet, err := parseRemoteNet(rNetStr)
	if err != nil {
		panic(err)
	}

	log.Printf("Network %s, mtu %d", cRemoteNet.str, mtu)

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
				panic(err)
			}
			err = conn.WriteMessage(websocket.BinaryMessage, packet[:n])
			if err != nil {
				panic(err)
			}
		}
	}()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			panic(err)
		}
		iface.Write(msg)
	}
}
