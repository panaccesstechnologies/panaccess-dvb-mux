package mpegts

import "fmt"

const PCRClockHz = 90000

type ContinuityTracker struct {
	last map[uint16]uint8
}

func NewContinuityTracker() *ContinuityTracker { return &ContinuityTracker{last: make(map[uint16]uint8)} }

// Observe returns true when a payload-bearing packet has an unexpected CC.
// Adaptation-only packets are ignored because their continuity counter need not
// advance. Duplicate packets are accepted separately by the caller if needed.
func (t *ContinuityTracker) Observe(p Packet) (discontinuity bool, err error) {
	if !p.HasPayload() { return false, nil }
	pid := p.PID(); cc := p.ContinuityCounter()
	if prev, ok := t.last[pid]; ok {
		expected := (prev + 1) & 0x0f
		if cc != expected { discontinuity = true }
	}
	t.last[pid] = cc
	return discontinuity, nil
}

func DecodePCR(adaptation []byte) (uint64, error) {
	if len(adaptation) < 7 { return 0, fmt.Errorf("PCR requires 6 bytes after adaptation flags") }
	if adaptation[0]&0x10 == 0 { return 0, fmt.Errorf("PCR flag not set") }
	b := adaptation[1:7]
	base := uint64(b[0])<<25 | uint64(b[1])<<17 | uint64(b[2])<<9 | uint64(b[3])<<1 | uint64(b[4]>>7)
	ext := uint64(b[4]&0x01)<<8 | uint64(b[5])
	return base*300 + ext, nil
}

func EncodePCR(pcr uint64) [6]byte {
	pcr &= (uint64(1)<<42)-1
	base := pcr / 300; ext := pcr % 300
	var b [6]byte
	b[0]=byte(base>>25); b[1]=byte(base>>17); b[2]=byte(base>>9); b[3]=byte(base>>1); b[4]=byte(base<<7)&0x80 | 0x7e | byte(ext>>8); b[5]=byte(ext)
	return b
}

func ExtractPCR(p Packet) (uint64, bool, error) {
	if !p.HasAdaptation() || p.Data[4] < 7 { return 0, false, nil }
	return DecodePCR(p.Data[4:])
}
