package main

import (
	"github.com/Doridian/wsvpn/client/cli"
	shared_cli "github.com/Doridian/wsvpn/shared/cli"
)

func main() {
	configPtr, printDefaultConfigPtr := shared_cli.LoadFlags("client.yml", "")
	cli.Main(configPtr, printDefaultConfigPtr)
}
