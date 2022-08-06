package main

import (
	"errors"
	"flag"
	"fmt"

	client_cli "github.com/Doridian/wsvpn/client/cli"
	server_cli "github.com/Doridian/wsvpn/server/cli"
	shared_cli "github.com/Doridian/wsvpn/shared/cli"
)

func main() {
	sidePtr := flag.String("side", "", "client or server")
	configPtr, printDefaultConfigPtr := shared_cli.LoadFlags("SIDE.yml", "Config file name (\"SIDE.yml\" means use either server.yml or client.yml)")
	flag.Parse()

	if *configPtr == "SIDE.yml" {
		configName := fmt.Sprintf("%s.yml", *sidePtr)
		configPtr = &configName
	}

	switch *sidePtr {
	case "client":
		client_cli.Main(configPtr, printDefaultConfigPtr)
	case "server":
		server_cli.Main(configPtr, printDefaultConfigPtr)
	default:
		panic(errors.New("please choose a valid side: client or server"))
	}
}
