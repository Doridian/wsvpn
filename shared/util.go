package shared

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"sync"
)

type MacAddr [6]byte

var DefaultMac = MacAddr{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}

func ExecCmd(cmd string, arg ...string) error {
	cmdO := exec.Command(cmd, arg...)
	cmdO.Stdout = os.Stdout
	cmdO.Stderr = os.Stderr
	return cmdO.Run()
}

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

func NetworkInterfaceExists(name string) bool {
	iface, err := net.InterfaceByName(name)
	if err != nil {
		return false
	}
	return iface != nil
}

func FindLowestNetworkInterfaceByPrefix(prefix string) string {
	i := 0
	var ifaceName string
	for {
		ifaceName = fmt.Sprintf("%s%d", prefix, i)
		if !NetworkInterfaceExists(ifaceName) {
			return ifaceName
		}
		i += 1
	}
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
	return mtu + 18
}

func MakeSimpleCond() *sync.Cond {
	return sync.NewCond(&sync.Mutex{})
}
