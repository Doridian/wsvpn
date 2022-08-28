package shared

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"sync"
)

func ExecCmd(cmd string, arg ...string) error {
	cmdO := exec.Command(cmd, arg...)
	cmdO.Stdout = os.Stdout
	cmdO.Stderr = os.Stderr
	err := cmdO.Run()
	if err == nil {
		return nil
	}
	return fmt.Errorf("command %s %s: %v", cmd, strings.Join(arg, " "), err)
}

func ExecCmdGetStdOut(cmd string, arg ...string) (string, error) {
	stdoutBuffer := &bytes.Buffer{}
	cmdO := exec.Command(cmd, arg...)
	cmdO.Stdout = bufio.NewWriter(stdoutBuffer)
	cmdO.Stderr = os.Stderr
	err := cmdO.Run()
	if err == nil {
		return stdoutBuffer.String(), nil
	}
	return "", fmt.Errorf("command %s %s: %v", cmd, strings.Join(arg, " "), err)
}

func IPNetGetNetMask(ipNet *net.IPNet) string {
	mask := ipNet.Mask
	return fmt.Sprintf("%d.%d.%d.%d", mask[0], mask[1], mask[2], mask[3])
}

func BoolToString(val bool, trueval string, falseval string) string {
	if val {
		return trueval
	}
	return falseval
}

func BoolIfString(val bool, trueval string) string {
	return BoolToString(val, trueval, "")
}

func BoolToEnabled(val bool) string {
	return BoolToString(val, "enabled", "disabled")
}

func GetPacketBufferSizeByMTU(mtu int) int {
	return mtu + 18 + 64
}

func MakeSimpleCond() *sync.Cond {
	return sync.NewCond(&sync.Mutex{})
}

var DefaultMac = net.HardwareAddr{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}
var DefaultIPv4 = net.IPv4(255, 255, 255, 255)
