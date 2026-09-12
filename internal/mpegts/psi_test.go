package mpegts

import (
	"bytes"
	"testing"
)

func TestPATRoundTrip(t *testing.T) {
	in := PAT{TSID: 1, Version: 3, Current: true, Programs: []PATProgram{{Number: 100, PMTPID: 0x100}, {Number: 200, PMTPID: 0x110}}}
	s, err := in.MarshalSection()
	if err != nil { t.Fatal(err) }
	out, err := ParsePAT(s)
	if err != nil { t.Fatal(err) }
	if out.TSID != in.TSID || out.Version != in.Version || len(out.Programs) != 2 { t.Fatalf("PAT mismatch: %+v", out) }
	if out.Programs[0] != in.Programs[0] || out.Programs[1] != in.Programs[1] { t.Fatalf("program mismatch: %+v", out.Programs) }
}

func TestPATNetworkPIDRoundTrip(t *testing.T) {
	in := PAT{TSID: 7, Version: 1, Current: true, Programs: []PATProgram{{Number: 0, NetworkPID: 0x10}, {Number: 100, PMTPID: 0x100}}}
	s, err := in.MarshalSection()
	if err != nil { t.Fatal(err) }
	out, err := ParsePAT(s)
	if err != nil { t.Fatal(err) }
	if len(out.Programs) != 2 || out.Programs[0].Number != 0 || out.Programs[0].NetworkPID != 0x10 { t.Fatalf("NIT entry mismatch: %+v", out.Programs) }
}

func TestPMTRoundTrip(t *testing.T) {
	in := PMT{ProgramNumber: 100, Version: 2, Current: true, PCRPID: 0x101, ProgramDescriptors: []byte{0x09, 0x02, 0x12, 0x34}, Streams: []PMTStream{{StreamType: 0x1b, PID: 0x101}, {StreamType: 0x0f, PID: 0x102, Descriptors: []byte{0x52, 0x01, 0x01}}}}
	s, err := in.MarshalSection()
	if err != nil { t.Fatal(err) }
	out, err := ParsePMT(s)
	if err != nil { t.Fatal(err) }
	if out.ProgramNumber != in.ProgramNumber || out.PCRPID != in.PCRPID || len(out.Streams) != 2 { t.Fatalf("PMT mismatch: %+v", out) }
	if !bytes.Equal(out.ProgramDescriptors, in.ProgramDescriptors) || !bytes.Equal(out.Streams[1].Descriptors, in.Streams[1].Descriptors) { t.Fatal("descriptor mismatch") }
}

func TestPacketizeSection(t *testing.T) {
	s := make([]byte, 184)
	for i := range s { s[i] = byte(i) }
	pkts, err := PacketizeSection(0x100, s, 14)
	if err != nil { t.Fatal(err) }
	if len(pkts) != 2 || !pkts[0].PUSI() || pkts[0].ContinuityCounter() != 14 || pkts[1].ContinuityCounter() != 15 { t.Fatalf("unexpected packets: %d", len(pkts)) }
	if pkts[0].Data[4] != 0 { t.Fatal("missing pointer field") }
}

func TestPCRRoundTrip(t *testing.T) {
	want := uint64(123456789)
	b := EncodePCR(want)
	a := append([]byte{7, 0x10}, b[:]...)
	got, err := DecodePCR(a)
	if err != nil { t.Fatal(err) }
	if got != want { t.Fatalf("PCR got %d want %d", got, want) }
}
