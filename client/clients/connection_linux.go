//go:build linux

package clients

import (
	"crypto/tls"
	"errors"
	"log"
	"net"
	"os"

	"golang.org/x/sys/unix"
)

const fwmarkIoctl int = 36 /* unix.SO_MARK */

var ErrUnknownConnType = errors.New("not a known conn type")

func setFirewallMark(conn net.Conn, mark int) error {
	var err error
	var file *os.File

	switch typedConn := conn.(type) {
	case *net.TCPConn:
		file, err = typedConn.File()
	case *net.UDPConn:
		file, err = typedConn.File()
	case *tls.Conn:
		return setFirewallMark(typedConn.NetConn(), mark)
	default:
		log.Printf("Unknown conn type: %T / %v", typedConn, typedConn)
		err = ErrUnknownConnType
	}

	if err != nil {
		return err
	}

	fd := file.Fd()
	return unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, fwmarkIoctl, mark)
}
