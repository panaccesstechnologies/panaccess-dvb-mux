package mpegts

import "testing"

func TestEMMInjector(t *testing.T) {
	inj, err := NewEMMInjector(0x1ffe, 12)
	if err != nil { t.Fatal(err) }
	section := []byte{0x82, 0xb0, 0x06, 0x00, 0x65, 0xc1, 0x00, 0x00, 0xaa, 0xbb, 0xcc}
	pkts, err := inj.InjectSection(section)
	if err != nil { t.Fatal(err) }
	if len(pkts) != 1 { t.Fatalf("got %d packets, want 1", len(pkts)) }
	p := pkts[0]
	if got := uint16(p.Data[1]&0x1f)<<8 | uint16(p.Data[2]); got != 0x1ffe { t.Fatalf("PID 0x%04x", got) }
	if p.Data[1]&0x40 == 0 { t.Fatal("PUSI not set") }
	if p.Data[3]&0x0f != 12 { t.Fatalf("CC=%d", p.Data[3]&0x0f) }
	if p.Data[4] != 0 { t.Fatalf("pointer=%d", p.Data[4]) }
	if got := p.Data[5:5+len(section)]; string(got) != string(section) { t.Fatalf("payload mismatch: %x", got) }
}

func TestEMMInjectorTSPacketMode(t *testing.T) {
	inj, err := NewEMMInjector(0x1ffe, 0)
	if err != nil { t.Fatal(err) }

	input := make([]byte, PacketSize)
	input[0] = SyncByte
	input[1] = 0x40 | 0x03 // PUSI + input PID 0x0003; head-end must replace PID.
	input[2] = 0x03
	input[3] = 0x13 // payload-only, continuity counter 3
	for i := 4; i < PacketSize; i++ {
		input[i] = byte(i)
	}

	pkts, err := inj.InjectTSPacket(input)
	if err != nil { t.Fatal(err) }
	if len(pkts) != 1 { t.Fatalf("got %d packets, want 1", len(pkts)) }

	p := pkts[0]
	if got := p.PID(); got != 0x1ffe { t.Fatalf("PID=0x%04x", got) }
	if !p.PUSI() { t.Fatal("PUSI not preserved") }
	if got := p.ContinuityCounter(); got != 3 { t.Fatalf("CC=%d", got) }
	for i := 3; i < PacketSize; i++ {
		if p.Data[i] != input[i] {
			t.Fatalf("byte %d changed: got 0x%02x want 0x%02x", i, p.Data[i], input[i])
		}
	}
}

func TestEMMInjectorTSPacketModeRejectsInvalidDatagrams(t *testing.T) {
	inj, err := NewEMMInjector(0x1ffe, 0)
	if err != nil { t.Fatal(err) }

	if _, err := inj.InjectTSPacket(make([]byte, PacketSize-1)); err == nil {
		t.Fatal("expected short TS packet to be rejected")
	}

	bad := make([]byte, PacketSize)
	bad[0] = 0x46
	if _, err := inj.InjectTSPacket(bad); err == nil {
		t.Fatal("expected invalid sync byte to be rejected")
	}
}
