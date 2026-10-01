package mpegts

import "testing"

func TestECMInjectorPacketizesAndAdvancesCC(t *testing.T) {
	inj, err := NewECMInjector(0x1ffe, 7)
	if err != nil { t.Fatal(err) }

	section := make([]byte, 200)
	section[0] = 0x80
	section[1] = 0xb0
	section[2] = 0xc5

	pkts, err := inj.InjectSection(section)
	if err != nil { t.Fatal(err) }
	if len(pkts) != 2 { t.Fatalf("got %d packets, want 2", len(pkts)) }
	if pkts[0].Data[0] != 0x47 || pkts[1].Data[0] != 0x47 { t.Fatal("invalid TS sync") }
	if got := uint16(pkts[0].Data[1]&0x1f)<<8 | uint16(pkts[0].Data[2]); got != 0x1ffe { t.Fatalf("PID 0x%04x, want 0x1ffe", got) }
	if pkts[0].Data[1]&0x40 == 0 { t.Fatal("first ECM packet missing PUSI") }
	if pkts[1].Data[1]&0x40 != 0 { t.Fatal("continuation ECM packet unexpectedly has PUSI") }
	if pkts[0].Data[3]&0x0f != 7 || pkts[1].Data[3]&0x0f != 8 { t.Fatal("bad ECM continuity counters") }
	if inj.CC != 9 { t.Fatalf("next CC=%d, want 9", inj.CC) }
}

func TestECMInjectorRejectsInvalidPID(t *testing.T) {
	if _, err := NewECMInjector(0x2000, 0); err == nil { t.Fatal("expected invalid PID error") }
}
