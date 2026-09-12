package transport

import (
	"fmt"
	"time"

	"github.com/panaccesstechnologies/panaccess-dvb-mux/internal/mpegts"
)

const DefaultBatchPackets = 7

// PacketQueue provides bounded back-pressure between transport I/O and future
// mux/scrambler workers. A buffered channel keeps Phase 2 deterministic while
// the production SPSC ring can replace this boundary without changing callers.
type PacketQueue struct {
	C chan mpegts.Packet
}

func NewPacketQueue(capacity int) (*PacketQueue, error) {
	if capacity < 1 { return nil, fmt.Errorf("queue capacity must be positive") }
	return &PacketQueue{C: make(chan mpegts.Packet, capacity)}, nil
}

// Pacer spaces transport batches according to a target MPEG-TS bitrate.
type Pacer struct {
	Bitrate uint64
	started time.Time
}

func NewPacer(bitrate uint64) (*Pacer, error) {
	if bitrate == 0 { return nil, fmt.Errorf("bitrate must be positive") }
	return &Pacer{Bitrate: bitrate}, nil
}

func (p *Pacer) Reset(now time.Time) { p.started = now }

// DelayForBytes returns the absolute target duration for a byte count. It does
// not sleep, allowing the caller to combine pacing with batching and shutdown.
func (p *Pacer) DelayForBytes(n int) time.Duration {
	if n <= 0 || p.Bitrate == 0 { return 0 }
	seconds := float64(n*8) / float64(p.Bitrate)
	return time.Duration(seconds * float64(time.Second))
}

// Wait sleeps until the next target point for a cumulative byte count.
func (p *Pacer) Wait(cumulativeBytes int) {
	if p.Bitrate == 0 || cumulativeBytes <= 0 { return }
	if p.started.IsZero() { p.started = time.Now() }
	target := p.started.Add(p.DelayForBytes(cumulativeBytes))
	if d := time.Until(target); d > 0 { time.Sleep(d) }
}

// Batch converts a packet slice into a compact TS datagram of up to maxPackets.
func Batch(packets []mpegts.Packet, maxPackets int) ([]byte, int, error) {
	if maxPackets < 1 { return nil, 0, fmt.Errorf("maxPackets must be positive") }
	if len(packets) == 0 { return nil, 0, nil }
	n := len(packets)
	if n > maxPackets { n = maxPackets }
	buf := make([]byte, n*mpegts.PacketSize)
	for i := 0; i < n; i++ { copy(buf[i*mpegts.PacketSize:], packets[i].Data[:]) }
	return buf, n, nil
}
