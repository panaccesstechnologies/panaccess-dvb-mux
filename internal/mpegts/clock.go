package mpegts

import "fmt"

const PCRClockHz = 90000

type ContinuityTracker struct { last map[uint16]uint8 }
func NewContinuityTracker() *ContinuityTracker { return &ContinuityTracker{last: make(map[uint16]uint8)} }

// Observe returns true when a payload-bearing packet has an unexpected CC.
func (t *ContinuityTracker) Observe(p Packet) (bool, error) {
	if !p.HasPayload() { return false, nil }
	pid, cc := p.PID(), p.ContinuityCounter()
	if prev, ok := t.last[pid]; ok && cc != ((prev+1)&0x0f) { t.last[pid] = cc; return true, nil }
	t.last[pid] = cc
	return false, nil
}

func DecodePCR(adaptation []byte) (uint64, error) {
	if len(adaptation) < 8 { return 0, fmt.Errorf("PCR requires adaptation length, flags and 6 PCR bytes") }
	if adaptation[0] < 7 || int(adaptation[0])+1 > len(adaptation) { return 0, fmt.Errorf("invalid adaptation field length") }
	if adaptation[1]&0x10 == 0 { return 0, fmt.Errorf("PCR flag not set") }
	b := adaptation[2:8]
	base := uint64(b[0])<<25 | uint64(b[1])<<17 | uint64(b[2])<<9 | uint64(b[3])<<1 | uint64(b[4]>>7)
	ext := uint64(b[4]&0x01)<<8 | uint64(b[5])
	return base*300 + ext, nil
}

func EncodePCR(pcr uint64) [6]byte {
	pcr &= (uint64(1)<<42)-1
	base, ext := pcr/300, pcr%300
	var b [6]byte
	b[0]=byte(base>>25); b[1]=byte(base>>17); b[2]=byte(base>>9); b[3]=byte(base>>1)
	b[4]=(byte(base)&1)<<7 | 0x7e | byte(ext>>8); b[5]=byte(ext)
	return b
}

func ExtractPCR(p Packet) (uint64, bool, error) {
	if !p.HasAdaptation() || p.Data[4] < 7 { return 0, false, nil }
	v, err := DecodePCR(p.Data[4:])
	if err != nil { return 0, false, err }
	return v, true, nil
}
