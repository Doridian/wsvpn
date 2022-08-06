package main

import (
	"flag"

	"github.com/Doridian/wsvpn/client/cli"
	shared_cli "github.com/Doridian/wsvpn/shared/cli"
)

func main() {
	configPtr, printDefaultConfigPtr := shared_cli.LoadFlags("client.yml", "")
	flag.Parse()

	cli.Main(configPtr, printDefaultConfigPtr)
}
