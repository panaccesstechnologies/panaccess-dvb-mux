package service

import (
	"testing"

	"github.com/panaccesstechnologies/panaccess-dvb-mux/internal/mpegts"
	"github.com/panaccesstechnologies/panaccess-dvb-mux/internal/simulcrypt"
)

func TestECMResponseIntegration(t *testing.T) {
	s := Service{
		ServiceID: 101,
		ECM: []CAEndpoint{{CASystemID: 0x4afc, PID: 0x1ffe}},
	}
	inserter, err := NewECMInserter(s, 0x4afc, 4)
	if err != nil { t.Fatal(err) }

	resp := simulcrypt.ECMResponse{
		ChannelID: 1,
		StreamID: 1,
		CPNumber: 10,
		ECMDatagram: []byte{0x80, 0xb0, 0x05, 0x00, 0x65, 0xc1, 0x00, 0x00},
	}
	pkts, err := inserter.InjectResponse(resp)
	if err != nil { t.Fatal(err) }
	if len(pkts) != 1 { t.Fatalf("got %d ECM packets, want 1", len(pkts)) }
	p := pkts[0]
	if p.Data[0] != 0x47 { t.Fatal("bad sync") }
	if got := uint16(p.Data[1]&0x1f)<<8 | uint16(p.Data[2]); got != 0x1ffe { t.Fatalf("PID 0x%04x, want 0x1ffe", got) }
	if p.Data[1]&0x40 == 0 { t.Fatal("missing PUSI") }
	if p.Data[3]&0x0f != 4 { t.Fatalf("CC=%d, want 4", p.Data[3]&0x0f) }
	if p.Data[4] != 0 { t.Fatalf("pointer=%d, want 0", p.Data[4]) }
	if got := p.Data[5:13]; string(got) != string(resp.ECMDatagram) { t.Fatalf("ECM payload mismatch: %x", got) }

	// Ensure this path is only control-plane injection; the source packet is not scrambled.
	if p.Data[3]&0xc0 != 0 { t.Fatal("ECM packet unexpectedly scrambled") }
	_ = mpegts.PacketSize
}
