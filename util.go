package main

import (
	"os"
	"os/exec"
)

func execCmd(cmd string, arg ...string) error {
	cmdO := exec.Command(cmd, arg...)
	cmdO.Stdout = os.Stdout
	cmdO.Stderr = os.Stderr
	return cmdO.Run()
}
