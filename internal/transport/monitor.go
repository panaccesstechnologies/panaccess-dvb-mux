package transport

import (
	"sync/atomic"

	"github.com/panaccesstechnologies/panaccess-dvb-mux/internal/mpegts"
)

// TSMonitor provides live transport-health counters for the Phase 2 runtime.
// It is intentionally independent of service routing and scrambling so later
// PSI/SI and CSA2 stages can reuse the same monitoring boundary.
type TSMonitor struct {
	ccTracker *mpegts.ContinuityTracker

	CCErrors     uint64
	PCRCount     uint64
	PCRErrors    uint64
	PCRBackwards uint64
	FirstPCR     uint64
	LastPCR      uint64
	firstSet     uint32
}

func NewTSMonitor() *TSMonitor {
	return &TSMonitor{ccTracker: mpegts.NewContinuityTracker()}
}

// Observe validates continuity for the packet PID and records PCR telemetry.
// A continuity error is counted but does not stop the transport path.
func (m *TSMonitor) Observe(p mpegts.Packet) {
	if m == nil {
		return
	}
	if bad, _ := m.ccTracker.Observe(p); bad {
		atomic.AddUint64(&m.CCErrors, 1)
	}

	pcr, ok, err := mpegts.ExtractPCR(p)
	if err != nil {
		atomic.AddUint64(&m.PCRErrors, 1)
		return
	}
	if !ok {
		return
	}
	atomic.AddUint64(&m.PCRCount, 1)
	if atomic.CompareAndSwapUint32(&m.firstSet, 0, 1) {
		atomic.StoreUint64(&m.FirstPCR, pcr)
		atomic.StoreUint64(&m.LastPCR, pcr)
		return
	}
	last := atomic.LoadUint64(&m.LastPCR)
	if pcr < last {
		atomic.AddUint64(&m.PCRBackwards, 1)
	}
	atomic.StoreUint64(&m.LastPCR, pcr)
}

// MonitorSnapshot is a point-in-time operational view of TS health.
type MonitorSnapshot struct {
	CCErrors     uint64
	PCRCount     uint64
	PCRErrors    uint64
	PCRBackwards uint64
	FirstPCR     uint64
	LastPCR      uint64
}

func (m *TSMonitor) Snapshot() MonitorSnapshot {
	if m == nil {
		return MonitorSnapshot{}
	}
	return MonitorSnapshot{
		CCErrors:     atomic.LoadUint64(&m.CCErrors),
		PCRCount:     atomic.LoadUint64(&m.PCRCount),
		PCRErrors:    atomic.LoadUint64(&m.PCRErrors),
		PCRBackwards: atomic.LoadUint64(&m.PCRBackwards),
		FirstPCR:     atomic.LoadUint64(&m.FirstPCR),
		LastPCR:      atomic.LoadUint64(&m.LastPCR),
	}
}
