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

func extendTAPConfig(config *water.Config) {

}
