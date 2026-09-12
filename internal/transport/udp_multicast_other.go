//go:build !(linux || darwin || freebsd || netbsd || openbsd)

package transport

import (
	"fmt"
	"net"
)

func setIPv4MulticastInterfaceFD(_ int, _ net.IP) error {
	return fmt.Errorf("explicit IPv4 multicast interface selection is not supported on this platform")
}
