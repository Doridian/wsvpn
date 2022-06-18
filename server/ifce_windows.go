//go:build windows

package main

import (
	"errors"
	"net"

	"github.com/songgao/water"
)

func configIface(dev *water.Interface, ipConfig bool, mtu int, ipClient net.IP, ipServer net.IP, subnet *net.IPNet) error {
	return errors.New("running the server on Windows is not supported at the moment")
}

func extendTAPConfig(config *water.Config) {

}

func extendTUNConfig(tunConfig *water.Config) {

}
