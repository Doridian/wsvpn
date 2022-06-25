package cli

import (
	"flag"
	"time"

	"github.com/Doridian/wsvpn/shared/sockets"
)

var pingIntervalPtr = flag.Duration("ping-interval", time.Second*time.Duration(30), "Send ping frames in this interval")
var pingTimeoutPtr = flag.Duration("ping-timeout", time.Second*time.Duration(5), "Disconnect if no ping response is received after timeout")

func UsePingFlags(sock *sockets.Socket) {
	sock.ConfigurePing(*pingIntervalPtr, *pingTimeoutPtr)
}

type PingFlagsSocketConfigurator struct {
}

func (c *PingFlagsSocketConfigurator) ConfigureSocket(sock *sockets.Socket) error {
	UsePingFlags(sock)
	return nil
}
