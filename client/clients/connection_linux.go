//go:build linux

package clients

import (
	"crypto/tls"
	"errors"
	"log"
	"net"
	"syscall"

	"golang.org/x/sys/unix"
)

const fwmarkIoctl int = 36 /* unix.SO_MARK */

var ErrUnknownConnType = errors.New("not a known conn type")

func setFirewallMark(conn net.Conn, mark int) error {
	var err error
	var syscallConn syscall.Conn

	switch typedConn := conn.(type) {
	case syscall.Conn:
		syscallConn = typedConn
	case *tls.Conn:
		return setFirewallMark(typedConn.NetConn(), mark)
	default:
		log.Printf("Unknown conn type: %T (%v)", typedConn, typedConn)
		err = ErrUnknownConnType
	}

	if err != nil {
		return err
	}

	var operr error
	fd, err := syscallConn.SyscallConn()
	if err != nil {
		return err
	}

	err = fd.Control(func(fd uintptr) {
		operr = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, fwmarkIoctl, int(mark))
	})

	if err == nil {
		return operr
	}

	return err
}
