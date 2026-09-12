package transport

import (
	"encoding/binary"
	"testing"

	"github.com/panaccesstechnologies/panaccess-dvb-mux/internal/mpegts"
)

func TestParseDatagram1316(t *testing.T) {
	buf := make([]byte, 7*mpegts.PacketSize)
	for i := 0; i < 7; i++ {
		off := i * mpegts.PacketSize
		buf[off] = mpegts.SyncByte
		buf[off+1] = byte(i)
		buf[off+2] = 1
		buf[off+3] = 0x10 | byte(i&0x0f)
	}
	packets, err := ParseDatagram(buf)
	if err != nil { t.Fatal(err) }
	if len(packets) != 7 { t.Fatalf("got %d packets", len(packets)) }
	if packets[6].PID() != 0x0601 { t.Fatalf("unexpected PID 0x%x", packets[6].PID()) }
}

func TestParseRTPMPEGTS(t *testing.T) {
	b := make([]byte, 12+188)
	b[0] = 0x80
	b[1] = 33
	binary.BigEndian.PutUint16(b[2:4], 42)
	binary.BigEndian.PutUint32(b[4:8], 90000)
	binary.BigEndian.PutUint32(b[8:12], 0x12345678)
	b[12] = mpegts.SyncByte
	p, err := ParseRTP(b)
	if err != nil { t.Fatal(err) }
	if err := ValidateMPEGTSRTP(p); err != nil { t.Fatal(err) }
	if p.PayloadType != 33 || p.SequenceNumber != 42 || p.SSRC != 0x12345678 { t.Fatalf("bad RTP header: %+v", p) }
}

func TestRTPSequenceTracker(t *testing.T) {
	var tr RTPSequenceTracker
	if gap, dup := tr.Observe(100); gap != 0 || dup { t.Fatal("first packet should sync") }
	if gap, dup := tr.Observe(101); gap != 0 || dup { t.Fatal("contiguous packet reported incorrectly") }
	if gap, dup := tr.Observe(101); !dup || gap != 0 { t.Fatal("duplicate not detected") }
	if gap, dup := tr.Observe(104); gap != 2 || dup { t.Fatalf("expected gap 2, got %d duplicate=%v", gap, dup) }
}

func TestBatch(t *testing.T) {
	packets := make([]mpegts.Packet, 8)
	for i := range packets { packets[i].Data[0] = mpegts.SyncByte }
	b, n, err := Batch(packets, 7)
	if err != nil { t.Fatal(err) }
	if n != 7 || len(b) != 7*188 { t.Fatalf("got %d packets / %d bytes", n, len(b)) }
}

func TestPacerDuration(t *testing.T) {
	p, err := NewPacer(8_000_000)
	if err != nil { t.Fatal(err) }
	if got := p.DelayForBytes(1000); got != 1*time.Millisecond { t.Fatalf("got %v", got) }
}
