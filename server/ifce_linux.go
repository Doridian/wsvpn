// +build linux
package main

import (
	"fmt"
	"github.com/Doridian/wsvpn/shared"
	"github.com/songgao/water"
	"net"
	"syscall"
)

func configIface(dev *water.Interface, ipConfig bool, mtu int, ipClient net.IP, ipServer net.IP) error {
	if !ipConfig {
		return shared.ExecCmd("ifconfig", dev.Name(), "mtu", fmt.Sprintf("%d", mtu), "up")
	}

	if dev.IsTAP() {
		return shared.ExecCmd("ifconfig", dev.Name(), ipServer.String(), "mtu", fmt.Sprintf("%d", mtu), "up")
	}
	return shared.ExecCmd("ifconfig", dev.Name(), ipServer.String(), "pointopoint", ipClient.String(), "mtu", fmt.Sprintf("%d", mtu), "up")
}

func setProcessUidGid(uid int, gid int) {
	err := syscall.Setresgid(gid, gid, gid)
	if err != nil {
		panic(err)
	}
	err = syscall.Setresuid(uid, uid, uid)
	if err != nil {
		panic(err)
	}
}
