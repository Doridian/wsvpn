package shared

import (
	"log"
)

var (
	Version         = "dev"
	ProtocolVersion = 10
)

func PrintVersion() {
	log.Printf("Local version is: %s (protocol %d)", Version, ProtocolVersion)
}
