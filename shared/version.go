package shared

import (
	"flag"
	"fmt"
	"log"
	"os"
)

var (
	Version         = "dev"
	ProtocolVersion = 1
)

func PrintVersion() {
	log.Printf("Local version is: %s (protocol %d)", Version, ProtocolVersion)
}

func UsageWithVersion() {
	fmt.Fprintf(flag.CommandLine.Output(), "WSVPN version %s\nUsage of %s:\n", Version, os.Args[0])
	flag.PrintDefaults()
}
