//go:build linux

package transport

import (
	"fmt"
	"net"
	"syscall"
)

func setIPv4MulticastInterfaceFD(fd int, ip net.IP) error {
	ip4 := ip.To4()
	if ip4 == nil {
		return fmt.Errorf("IPv4 multicast interface address is required")
	}
	var addr [4]byte
	copy(addr[:], ip4)
	return syscall.SetsockoptInet4Addr(fd, syscall.IPPROTO_IP, syscall.IP_MULTICAST_IF, addr)
}
