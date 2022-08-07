package shared

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"sync"
)

type MacAddr [6]byte
type IPv4 = [4]byte

var DefaultMac = MacAddr{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}
var DefaultIPv4 = IPv4{255, 255, 255, 255}

type EtherType = uint16

const (
	ETHTYPE_IPV4 = 0x0800
	ETHTYPE_ARP  = 0x0806
	ETHTYPE_IPV6 = 0x86DD
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

func GetEtherType(packet []byte) EtherType {
	return uint16(packet[12])<<8 | uint16(packet[13])
}

func GetSrcIPv4(packet []byte, offset int) IPv4 {
	var ip IPv4
	copy(ip[:], packet[12+offset:16+offset])
	return ip
}

func GetDestIPv4(packet []byte, offset int) IPv4 {
	var ip IPv4
	copy(ip[:], packet[16+offset:20+offset])
	return ip
}

func NetIPToIPv4(ip net.IP) IPv4 {
	var res IPv4
	copy(res[:], ip[0:4])
	return res
}

func IPv4IsUnicast(ip IPv4) bool {
	if ip == DefaultIPv4 {
		return false
	}
	if ip[0]&0xf0 == 0xe0 {
		return false
	}
	return true
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
