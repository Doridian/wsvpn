// +build darwin
package main

import (
	"os/exec"
)

func configIface(dev string, ipClient string, ipServer string)  error {
	err = exec.Command("ifconfig", dev, "mtu", "1280").Run()
	if err != nil {
		return err
	}
	err := exec.Command("ifconfig", dev, ipServer, ipClient, "up").Run()
	if err != nil {
		return err
	}
	return nil
}
