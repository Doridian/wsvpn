package main

import (
	"encoding/base64"
	"errors"
	"flag"
	"github.com/Doridian/wsvpn/shared"
	"github.com/gorilla/websocket"
	"github.com/songgao/water"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const DEFAULT_URL = "ws://example.com"

var connectAddr = flag.String("connect", DEFAULT_URL, "Server address to connect to")
var authFile = flag.String("auth-file", "", "File to read authentication from in the format user:password")
var upScript = flag.String("up-script", "", "Script to run once the VPN is online")
var downScript = flag.String("down-script", "", "Script to run when the VPN goes offline")

func productionWarnings(str string) {
	for n := 0; n <= 5; n++ {
		log.Printf("DO NOT USE THIS IN PRODUCTION! %s!", str)
	}
}

func runEventScript(script *string, op string, cRemoteNet *remoteNet, iface *water.Interface) error {
	if script == nil {
		return nil
	}
	scriptStr := *script
	if scriptStr == "" {
		return nil
	}

	return shared.ExecCmd(scriptStr, op, cRemoteNet.str, iface.Name())
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

	var iface *water.Interface
	var cRemoteNet *remoteNet

	defer func() {
		if iface != nil {
			runEventScript(downScript, "down", cRemoteNet, iface)
			iface.Close()
		}
	}()

	socket := shared.MakeSocket("0", conn, nil)
	socket.AddCommandHandler("addroute", func(args []string) error {
		if iface == nil || cRemoteNet == nil {
			return errors.New("Cannot addroute before init")
		}

		if len(args) < 1 {
			return errors.New("addroute needs 1 argument")
		}
		_, routeNet, err := net.ParseCIDR(args[0])
		if err != nil {
			return err
		}
		return addRoute(iface, cRemoteNet, routeNet)
	})
	socket.AddCommandHandler("init", func(args []string) error {
		var err error

		rNetStr := args[0]
		mtu, err := strconv.Atoi(args[1])
		if err != nil {
			panic(err)
		}

		cRemoteNet, err = parseRemoteNet(rNetStr)
		if err != nil {
			panic(err)
		}

		log.Printf("Network %s, mtu %d", cRemoteNet.str, mtu)

		ifconfig := getPlatformSpecifics(cRemoteNet, mtu, water.Config{
			DeviceType: water.TUN,
		})
		iface, err = water.New(ifconfig)
		if err != nil {
			panic(err)
		}

		log.Printf("Opened %s", iface.Name())

		err = configIface(iface, cRemoteNet, mtu, false)
		if err != nil {
			panic(err)
		}

		log.Printf("Configured interface. Starting operations.")
		err = socket.SetInterface(iface)
		if err != nil {
			panic(err)
		}

		runEventScript(upScript, "up", cRemoteNet, iface)

		return nil
	})
	socket.Serve()
	socket.Wait()
}
