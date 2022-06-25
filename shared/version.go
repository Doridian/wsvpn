package shared

import (
	"log"
)

var (
	Version         = "dev"
	ProtocolVersion = 3
)

func PrintVersion(prefix string) {
	log.Printf("[%s] Local version is: %s (protocol %d)", prefix, Version, ProtocolVersion)
}
