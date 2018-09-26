// +build darwin
package main

import (
	"fmt"
	"github.com/Doridian/wsvpn/shared"
	"github.com/songgao/water"
	"net"
	"syscall"
)

func configIface(dev *water.Interface, ipConfig bool, mtu int, ipClient net.IP, ipServer net.IP) error {
	err := shared.ExecCmd("ifconfig", dev.Name(), "mtu", fmt.Sprintf("%d", mtu))
	if err != nil {
		return err
	}

	if !ipConfig {
		return shared.ExecCmd("ifconfig", dev.Name(), "up")
	}

	if dev.IsTAP() {
		return shared.ExecCmd("ifconfig", dev.Name(), ipServer.String(), "up")
	}
	return shared.ExecCmd("ifconfig", dev.Name(), ipServer.String(), ipClient.String(), "up")
}

func setProcessUidGid(uid int, gid int) {
	err := syscall.Setregid(gid, gid)
	if err != nil {
		panic(err)
	}
	err = syscall.Setreuid(uid, uid)
	if err != nil {
		panic(err)
	}
}
