package main

import (
	"encoding/base64"
	"flag"
	"github.com/gorilla/websocket"
	"github.com/songgao/water"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const DEFAULT_URL = "ws://user:password@example.com"

var connectAddr = flag.String("connect", DEFAULT_URL, "Server address to connect to")
var authFile = flag.String("auth-file", "", "File to read authentication from in the format user:password")

func productionWarnings(str string) {
	for n := 0; n <= 5; n++ {
		log.Printf("DO NOT USE THIS IN PRODUCTION! %s!", str)
	}
}

func main() {
	flag.Parse()

	destUrlString := *connectAddr
	if destUrlString == DEFAULT_URL {
		flag.PrintDefaults()
		return
	}

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
		authDataSplit := strings.SplitN(string(authData), ":", 2)
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
		productionWarnings("PASSWORD ON THE COMMAND LINE")
	}

	header := http.Header{}
	if userInfo != nil {
		log.Printf("Connecting to %s as user %s.", dest.String(), userInfo.Username())
		if _, pws := userInfo.Password(); !pws {
			productionWarnings("NO PASSWORD SET")
		}
		header.Add("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(userInfo.String())))
	} else {
		log.Printf("Connecting to %s without authentication.", dest.String())
		productionWarnings("NO AUTHENTICATION SET")
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
