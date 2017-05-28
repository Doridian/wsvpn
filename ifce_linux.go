// +build linux
package main

import (
	"os/exec"
)

func configIface(dev string, ipClient string, ipServer string)  error {
	err := exec.Command("ifconfig", dev, ipServer, "pointopoint", ipClient, "up").Run()
	if err != nil {
		return err
	}
	err = exec.Command("ifconfig", dev, "up").Run()
	if err != nil {
		return err
	}
	return nil
}
