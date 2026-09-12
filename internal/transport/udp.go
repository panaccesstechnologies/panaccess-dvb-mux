package transport

import (
	"fmt"
	"net"
	"time"

	"github.com/panaccesstechnologies/panaccess-dvb-mux/internal/mpegts"
)

const DefaultDatagramSize = 2048

// UDPSource receives MPEG-TS datagrams. It accepts one or more complete
// 188-byte packets per datagram, including the common 7-packet/1316-byte form.
type UDPSource struct {
	conn *net.UDPConn
}

func ListenUDP(bindAddr string, port int, multicastGroup string, iface *net.Interface) (*UDPSource, error) {
	if port < 1 || port > 65535 {
		return nil, fmt.Errorf("invalid UDP port %d", port)
	}
	addr := &net.UDPAddr{IP: net.ParseIP(bindAddr), Port: port}
	if addr.IP == nil {
		return nil, fmt.Errorf("invalid bind address %q", bindAddr)
	}
	if multicastGroup != "" {
		group := net.ParseIP(multicastGroup)
		if group == nil || !group.IsMulticast() {
			return nil, fmt.Errorf("invalid multicast group %q", multicastGroup)
		}
		addr.IP = group
		conn, err := net.ListenMulticastUDP("udp", iface, addr)
		if err != nil {
			return nil, fmt.Errorf("listen multicast %s:%d: %w", multicastGroup, port, err)
		}
		return &UDPSource{conn: conn}, nil
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, fmt.Errorf("listen UDP %s:%d: %w", bindAddr, port, err)
	}
	return &UDPSource{conn: conn}, nil
}

func (s *UDPSource) Close() error {
	if s == nil || s.conn == nil {
		return nil
	}
	return s.conn.Close()
}

// ReadDatagram reads one UDP datagram without assuming a particular TS packet
// aggregation. The returned slice aliases the supplied receive buffer.
func (s *UDPSource) ReadDatagram(buf []byte) ([]byte, *net.UDPAddr, error) {
	if len(buf) < mpegts.PacketSize {
		return nil, nil, fmt.Errorf("receive buffer too small: %d", len(buf))
	}
	n, addr, err := s.conn.ReadFromUDP(buf)
	if err != nil {
		return nil, nil, err
	}
	if n == 0 || n%mpegts.PacketSize != 0 {
		return nil, addr, fmt.Errorf("UDP datagram has %d bytes, not a multiple of 188", n)
	}
	return buf[:n], addr, nil
}

func (s *UDPSource) SetReadDeadline(t time.Time) error {
	return s.conn.SetReadDeadline(t)
}

// ParseDatagram validates and splits a UDP datagram into fixed-size TS packets.
func ParseDatagram(datagram []byte) ([]mpegts.Packet, error) {
	if len(datagram) == 0 || len(datagram)%mpegts.PacketSize != 0 {
		return nil, fmt.Errorf("TS datagram length %d is not a multiple of 188", len(datagram))
	}
	packets := make([]mpegts.Packet, 0, len(datagram)/mpegts.PacketSize)
	for off := 0; off < len(datagram); off += mpegts.PacketSize {
		p, err := mpegts.ParsePacket(datagram[off : off+mpegts.PacketSize])
		if err != nil {
			return nil, fmt.Errorf("TS packet at datagram offset %d: %w", off, err)
		}
		packets = append(packets, p)
	}
	return packets, nil
}
