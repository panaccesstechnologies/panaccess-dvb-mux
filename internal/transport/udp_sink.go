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
}

func NewUDPSink(localAddr, remoteAddr string) (*UDPSink, error) {
	remote, err := net.ResolveUDPAddr("udp", remoteAddr)
	if err != nil {
		return nil, fmt.Errorf("resolve UDP destination %q: %w", remoteAddr, err)
	}
	if remote.IP == nil {
		return nil, fmt.Errorf("UDP destination has no IP: %q", remoteAddr)
	}
	var local *net.UDPAddr
	if localAddr != "" {
		local, err = net.ResolveUDPAddr("udp", localAddr)
		if err != nil {
			return nil, fmt.Errorf("resolve UDP local address %q: %w", localAddr, err)
		}
	}
	conn, err := net.DialUDP("udp", local, remote)
	if err != nil {
		return nil, fmt.Errorf("open UDP sink: %w", err)
	}
	return &UDPSink{conn: conn, target: remote}, nil
}

func (s *UDPSink) Close() error {
	if s == nil || s.conn == nil { return nil }
	return s.conn.Close()
}

// WritePackets sends packets in one UDP datagram. maxDatagram must be a
// multiple of 188; 1316 (7 TS packets) is a common MTU-safe aggregation.
func (s *UDPSink) WritePackets(packets []mpegts.Packet, maxDatagram int) (int, error) {
	if maxDatagram < 188 || maxDatagram%188 != 0 {
		return 0, fmt.Errorf("maxDatagram must be a positive multiple of 188")
	}
	if len(packets) == 0 { return 0, nil }
	maxPackets := maxDatagram / 188
	written := 0
	for written < len(packets) {
		n := len(packets) - written
		if n > maxPackets { n = maxPackets }
		buf := make([]byte, n*188)
		for i := 0; i < n; i++ { copy(buf[i*188:], packets[written+i].Data[:]) }
		if _, err := s.conn.Write(buf); err != nil {
			return written, fmt.Errorf("write UDP MPEG-TS: %w", err)
		}
		written += n
	}
	return written, nil
}
