package service

import (
	"testing"

	"github.com/panaccesstechnologies/panaccess-dvb-mux/internal/mpegts"
)

func TestEMMDatagramIntegration(t *testing.T) {
	s := Service{ServiceID: 101, EMM: []CAEndpoint{{CASystemID: 0x4afc, PID: 0x1ffe}}}
	inserter, err := NewEMMInserter(s, 0x4afc, 5)
	if err != nil { t.Fatal(err) }

	datagram := []byte{0x82, 0xb0, 0x06, 0x00, 0x65, 0xc1, 0x00, 0x00, 0xaa, 0xbb, 0xcc}
	pkts, err := inserter.InjectDatagram(datagram)
	if err != nil { t.Fatal(err) }
	if len(pkts) != 1 { t.Fatalf("got %d packets, want 1", len(pkts)) }
	p := pkts[0]
	if got := uint16(p.Data[1]&0x1f)<<8 | uint16(p.Data[2]); got != 0x1ffe { t.Fatalf("PID 0x%04x", got) }
	if p.Data[1]&0x40 == 0 { t.Fatal("PUSI not set") }
	if p.Data[3]&0x0f != 5 { t.Fatalf("CC=%d", p.Data[3]&0x0f) }
	if p.Data[4] != 0 { t.Fatalf("pointer=%d", p.Data[4]) }
	if got := p.Data[5:5+len(datagram)]; string(got) != string(datagram) { t.Fatalf("EMM mismatch: %x", got) }
	_ = mpegts.PacketSize
}
