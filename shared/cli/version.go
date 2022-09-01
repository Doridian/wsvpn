package cli

import (
	"flag"
	"fmt"
	"os"

	"github.com/Doridian/wsvpn/shared"
)

func UsageWithVersion() {
	_, _ = fmt.Fprintf(flag.CommandLine.Output(), "WSVPN version %s\nUsage of %s:\n", shared.Version, os.Args[0])
	flag.PrintDefaults()
}
