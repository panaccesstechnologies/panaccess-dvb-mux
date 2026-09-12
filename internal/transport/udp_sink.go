package transport

import (
	"fmt"
	"net"

	"github.com/panaccesstechnologies/panaccess-dvb-mux/internal/mpegts"
)

// UDPSink emits aggregated MPEG-TS packets to a unicast or multicast UDP endpoint.
type UDPSink struct {
	conn   *net.UDPConn
	target *net.UDPAddr
	iface  *net.Interface
}

// NewUDPSink preserves the original API. When no interface is specified,
// normal kernel routing selects the outgoing interface.
func NewUDPSink(localAddr, remoteAddr string) (*UDPSink, error) {
	return newUDPSink(localAddr, remoteAddr, "")
}

// NewUDPSinkWithInterface creates a sink whose output is explicitly associated
// with the named network interface. For unicast, the interface's IPv4 address
// is used as the local source address. For multicast, the implementation uses
// the multicast socket's IP_MULTICAST_IF setting through the socket control API.
func NewUDPSinkWithInterface(localAddr, remoteAddr, interfaceName string) (*UDPSink, error) {
	if interfaceName == "" {
		return nil, fmt.Errorf("UDP egress interface name is required")
	}
	return newUDPSink(localAddr, remoteAddr, interfaceName)
}

func newUDPSink(localAddr, remoteAddr, interfaceName string) (*UDPSink, error) {
	remote, err := net.ResolveUDPAddr("udp", remoteAddr)
	if err != nil {
		return nil, fmt.Errorf("resolve UDP destination %q: %w", remoteAddr, err)
	}
	if remote.IP == nil {
		return nil, fmt.Errorf("UDP destination has no IP: %q", remoteAddr)
	}

	var iface *net.Interface
	var local *net.UDPAddr
	if interfaceName != "" {
		iface, err = net.InterfaceByName(interfaceName)
		if err != nil {
			return nil, fmt.Errorf("resolve UDP egress interface %q: %w", interfaceName, err)
		}
		if iface.Flags&net.FlagUp == 0 {
			return nil, fmt.Errorf("UDP egress interface %q is down", interfaceName)
		}
	}

	if localAddr != "" {
		local, err = net.ResolveUDPAddr("udp", localAddr)
		if err != nil {
			return nil, fmt.Errorf("resolve UDP local address %q: %w", localAddr, err)
		}
	} else if iface != nil {
		ip, err := interfaceIPv4(iface)
		if err != nil {
			return nil, err
		}
		local = &net.UDPAddr{IP: ip}
	}

	conn, err := net.DialUDP("udp", local, remote)
	if err != nil {
		return nil, fmt.Errorf("open UDP sink: %w", err)
	}

	if iface != nil && remote.IP.IsMulticast() {
		if err := setIPv4MulticastInterface(conn, iface); err != nil {
			_ = conn.Close()
			return nil, fmt.Errorf("set UDP multicast egress interface %q: %w", interfaceName, err)
		}
	}
	return &UDPSink{conn: conn, target: remote, iface: iface}, nil
}

func interfaceIPv4(iface *net.Interface) (net.IP, error) {
	addrs, err := iface.Addrs()
	if err != nil {
		return nil, fmt.Errorf("list addresses for UDP egress interface %q: %w", iface.Name, err)
	}
	for _, addr := range addrs {
		var ip net.IP
		switch v := addr.(type) {
		case *net.IPNet:
			ip = v.IP
		case *net.IPAddr:
			ip = v.IP
		}
		if ip4 := ip.To4(); ip4 != nil {
			return ip4, nil
		}
	}
	return nil, fmt.Errorf("UDP egress interface %q has no IPv4 address", iface.Name)
}

// setIPv4MulticastInterface selects the IPv4 interface used for multicast
// output without relying on UDPConn.SetMulticastInterface, which is not
// available in the Go networking API used by this repository.
func setIPv4MulticastInterface(conn *net.UDPConn, iface *net.Interface) error {
	ip, err := interfaceIPv4(iface)
	if err != nil {
		return err
	}
	pc, err := conn.SyscallConn()
	if err != nil {
		return fmt.Errorf("get UDP socket control: %w", err)
	}
	var controlErr error
	if err := pc.Control(func(fd uintptr) {
		controlErr = setIPv4MulticastInterfaceFD(int(fd), ip)
	}); err != nil {
		return err
	}
	return controlErr
}

func (s *UDPSink) Close() error {
	if s == nil || s.conn == nil {
		return nil
	}
	return s.conn.Close()
}

// WritePackets sends packets in one UDP datagram. maxDatagram must be a
// multiple of 188; 1316 (7 TS packets) is a common MTU-safe aggregation.
func (s *UDPSink) WritePackets(packets []mpegts.Packet, maxDatagram int) (int, error) {
	if maxDatagram < 188 || maxDatagram%188 != 0 {
		return 0, fmt.Errorf("maxDatagram must be a positive multiple of 188")
	}
	if len(packets) == 0 {
		return 0, nil
	}
	maxPackets := maxDatagram / 188
	written := 0
	for written < len(packets) {
		n := len(packets) - written
		if n > maxPackets {
			n = maxPackets
		}
		buf := make([]byte, n*188)
		for i := 0; i < n; i++ {
			copy(buf[i*188:], packets[written+i].Data[:])
		}
		if _, err := s.conn.Write(buf); err != nil {
			return written, fmt.Errorf("write UDP MPEG-TS: %w", err)
		}
		written += n
	}
	return written, nil
}
