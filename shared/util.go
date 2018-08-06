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

type MacAddr [6]byte

func GetSrcMAC(packet []byte) MacAddr {
	var mac MacAddr
	copy(mac[:], packet[6:12])
	return mac
}

func GetDestMAC(packet []byte) MacAddr {
	var mac MacAddr
	copy(mac[:], packet[0:6])
	return mac
}

func MACIsUnicast(mac MacAddr) bool {
	return (mac[0] & 1) == 0
}
