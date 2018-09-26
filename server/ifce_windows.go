// +build windows
package main

import (
	"errors"
	"github.com/songgao/water"
	"net"
)

func configIface(dev *water.Interface, ipConfig bool, mtu int, ipClient net.IP, ipServer net.IP) error {
	return errors.New("Windows is not supported atm")
}

func setProcessUidGid(uid int, gid int) {
	log.Printf("setuid/setgid not supported on windows")
}
