package cli

import (
	"flag"
	"fmt"
	"os"

	"github.com/Doridian/wsvpn/shared"
)

func UsageWithVersion() {
	fmt.Fprintf(flag.CommandLine.Output(), "WSVPN version %s\nUsage of %s:\n", shared.Version, os.Args[0])
	flag.PrintDefaults()
}
