package shared

import (
	"flag"
	"fmt"
	"log"
	"os"
)

var (
	Version         = "dev"
	ProtocolVersion = 2
)

func PrintVersion(prefix string) {
	log.Printf("[%s] Local version is: %s (protocol %d)", prefix, Version, ProtocolVersion)
}

func UsageWithVersion() {
	fmt.Fprintf(flag.CommandLine.Output(), "WSVPN version %s\nUsage of %s:\n", Version, os.Args[0])
	flag.PrintDefaults()
}
