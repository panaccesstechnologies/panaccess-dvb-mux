package transport

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/panaccesstechnologies/panaccess-dvb-mux/internal/mpegts"
)

// StreamStats is safe to update from an ingest goroutine and read by a future
// control/API layer without taking the transport data path lock.
type StreamStats struct {
	PacketsReceived uint64
	BytesReceived   uint64
	Datagrams       uint64
	RTPPackets      uint64
	PacketErrors    uint64
	CCErrors        uint64
	RTPGaps         uint64
}

func (s *StreamStats) AddPackets(n uint64) { atomic.AddUint64(&s.PacketsReceived, n) }
func (s *StreamStats) AddBytes(n uint64) { atomic.AddUint64(&s.BytesReceived, n) }
func (s *StreamStats) AddDatagrams(n uint64) { atomic.AddUint64(&s.Datagrams, n) }
func (s *StreamStats) AddRTPPackets(n uint64) { atomic.AddUint64(&s.RTPPackets, n) }
func (s *StreamStats) AddPacketErrors(n uint64) { atomic.AddUint64(&s.PacketErrors, n) }
func (s *StreamStats) AddCCErrors(n uint64) { atomic.AddUint64(&s.CCErrors, n) }
func (s *StreamStats) AddRTPGaps(n uint64) { atomic.AddUint64(&s.RTPGaps, n) }

// Snapshot returns a coherent-enough operational view using atomic loads.
func (s *StreamStats) Snapshot() StreamStats {
	return StreamStats{
		PacketsReceived: atomic.LoadUint64(&s.PacketsReceived),
		BytesReceived: atomic.LoadUint64(&s.BytesReceived),
		Datagrams: atomic.LoadUint64(&s.Datagrams),
		RTPPackets: atomic.LoadUint64(&s.RTPPackets),
		PacketErrors: atomic.LoadUint64(&s.PacketErrors),
		CCErrors: atomic.LoadUint64(&s.CCErrors),
		RTPGaps: atomic.LoadUint64(&s.RTPGaps),
	}
}

// Stream is the Phase 2 transport identity used by later routing and mux layers.
type Stream struct {
	ID       string
	Source   string
	Protocol string // "udp" or "rtp"
	Queue    *PacketQueue
	Stats    StreamStats
	Pacer    *Pacer
}

func NewStream(id, source, protocol string, queueCapacity int, bitrate uint64) (*Stream, error) {
	if id == "" { return nil, fmt.Errorf("stream ID is required") }
	if protocol != "udp" && protocol != "rtp" { return nil, fmt.Errorf("unsupported protocol %q", protocol) }
	q, err := NewPacketQueue(queueCapacity)
	if err != nil { return nil, err }
	p, err := NewPacer(bitrate)
	if err != nil { return nil, err }
	return &Stream{ID: id, Source: source, Protocol: protocol, Queue: q, Pacer: p}, nil
}

// EnqueueDatagram validates a raw UDP/RTP datagram and places its TS packets on
// the bounded queue. It intentionally keeps transport parsing separate from
// future service routing and scrambling.
func (s *Stream) EnqueueDatagram(datagram []byte) error {
	if s.Protocol == "udp" {
		packets, err := ParseDatagram(datagram)
		if err != nil { s.Stats.AddPacketErrors(1); return err }
		for _, p := range packets { s.Queue.C <- p }
		s.Stats.AddDatagrams(1)
		s.Stats.AddPackets(uint64(len(packets)))
		s.Stats.AddBytes(uint64(len(datagram)))
		return nil
	}
	rtp, err := ParseRTP(datagram)
	if err != nil { s.Stats.AddPacketErrors(1); return err }
	if err := ValidateMPEGTSRTP(rtp); err != nil { s.Stats.AddPacketErrors(1); return err }
	packets, err := ParseDatagram(rtp.Payload)
	if err != nil { s.Stats.AddPacketErrors(1); return err }
	for _, p := range packets { s.Queue.C <- p }
	s.Stats.AddDatagrams(1)
	s.Stats.AddRTPPackets(1)
	s.Stats.AddPackets(uint64(len(packets)))
	s.Stats.AddBytes(uint64(len(datagram)))
	return nil
}

// PaceBatch applies target bitrate timing after a batch has been accounted for.
func (s *Stream) PaceBatch(bytes int) {
	if s.Pacer == nil || bytes <= 0 { return }
	s.Pacer.Wait(bytes)
}

func (s *Stream) ResetPacing(now time.Time) {
	if s.Pacer != nil { s.Pacer.Reset(now) }
}

// Ensure Packet remains the transport queue element type.
var _ mpegts.Packet
