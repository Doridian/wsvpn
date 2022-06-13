//go:build windows

package main

import (
	"errors"
	"github.com/songgao/water"
	"net"
)

func configIface(dev *water.Interface, ipConfig bool, mtu int, ipClient net.IP, ipServer net.IP, subnet *net.IPNet) error {
	return errors.New("Running the server on Windows is not supported at the moment")
}

func extendTAPConfig(config *water.Config) {

}
