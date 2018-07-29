package shared

import (
	"os"
	"os/exec"
)

func ExecCmd(cmd string, arg ...string) error {
	cmdO := exec.Command(cmd, arg...)
	cmdO.Stdout = os.Stdout
	cmdO.Stderr = os.Stderr
	return cmdO.Run()
}
