package cli

import (
	"os"
	"os/signal"
	"syscall"
)

func RegisterShutdownSignals(callback func()) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigs
		callback()
	}()
}
