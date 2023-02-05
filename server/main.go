package main

import (
	"github.com/Doridian/wsvpn/server/cli"
	shared_cli "github.com/Doridian/wsvpn/shared/cli"
)

func main() {
	configPtr, printDefaultConfigPtr := shared_cli.LoadFlags("server.yml", "")
	cli.Main(configPtr, printDefaultConfigPtr)
}
