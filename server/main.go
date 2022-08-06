package main

import (
	"flag"

	"github.com/Doridian/wsvpn/server/cli"
	shared_cli "github.com/Doridian/wsvpn/shared/cli"
)

func main() {
	configPtr, printDefaultConfigPtr := shared_cli.LoadFlags("server.yml", "")
	flag.Parse()

	cli.Main(configPtr, printDefaultConfigPtr)
}
