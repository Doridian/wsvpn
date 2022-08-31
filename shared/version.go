package shared

import (
	"log"
)

var (
	Version         = "dev"
	ProtocolVersion = 12
)

func PrintVersion() {
	log.Printf("Local version is: %s (protocol %d)", Version, ProtocolVersion)
}
