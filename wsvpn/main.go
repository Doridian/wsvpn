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
	modePtr := flag.String("mode", "", "client or server")
	configPtr, printDefaultConfigPtr := shared_cli.LoadFlags("MODE.yml", "Config file name (\"MODE.yml\" means use either server.yml or client.yml)")

	if *configPtr == "MODE.yml" {
		configName := fmt.Sprintf("%s.yml", *modePtr)
		configPtr = &configName
	}

	switch *modePtr {
	case "client":
		client_cli.Main(configPtr, printDefaultConfigPtr)
	case "server":
		server_cli.Main(configPtr, printDefaultConfigPtr)
	default:
		panic(errors.New("please choose a valid mode: client or server"))
	}
}
