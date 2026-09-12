package transport

import (
	"testing"

	"github.com/panaccesstechnologies/panaccess-dvb-mux/internal/mpegts"
)

func makeMonitorPacket(pid uint16, cc uint8, pcr *uint64) mpegts.Packet {
	var p mpegts.Packet
	p.Data[0] = mpegts.SyncByte
	p.Data[1] = byte(pid >> 8)
	p.Data[2] = byte(pid)
	p.Data[3] = 0x10 | (cc & 0x0f) // payload only
	if pcr != nil {
		p.Data[3] = 0x30 | (cc & 0x0f) // adaptation + payload
		p.Data[4] = 7                    // flags + 6 PCR bytes
		p.Data[5] = 0x10                 // PCR flag
		encoded := mpegts.EncodePCR(*pcr)
		copy(p.Data[6:12], encoded[:])
	}
	return p
}

func TestTSMonitorContinuityAndPCR(t *testing.T) {
	m := NewTSMonitor()
	pcr1 := uint64(90000 * 300)
	pcr2 := pcr1 + 3000

	m.Observe(makeMonitorPacket(0x0101, 0, &pcr1))
	m.Observe(makeMonitorPacket(0x0101, 1, &pcr2))
	m.Observe(makeMonitorPacket(0x0101, 3, nil))

	got := m.Snapshot()
	if got.CCErrors != 1 {
		t.Fatalf("CCErrors=%d, want 1", got.CCErrors)
	}
	if got.PCRCount != 2 {
		t.Fatalf("PCRCount=%d, want 2", got.PCRCount)
	}
	if got.PCRErrors != 0 {
		t.Fatalf("PCRErrors=%d, want 0", got.PCRErrors)
	}
	if got.PCRBackwards != 0 {
		t.Fatalf("PCRBackwards=%d, want 0", got.PCRBackwards)
	}
	if got.FirstPCR != pcr1 || got.LastPCR != pcr2 {
		t.Fatalf("PCR range=%d..%d, want %d..%d", got.FirstPCR, got.LastPCR, pcr1, pcr2)
	}
}

func TestTSMonitorPCRBackwards(t *testing.T) {
	m := NewTSMonitor()
	pcr1 := uint64(90000)
	pcr2 := uint64(45000)

	m.Observe(makeMonitorPacket(0x0101, 0, &pcr1))
	m.Observe(makeMonitorPacket(0x0101, 1, &pcr2))

	got := m.Snapshot()
	if got.PCRBackwards != 1 {
		t.Fatalf("PCRBackwards=%d, want 1", got.PCRBackwards)
	}
}
