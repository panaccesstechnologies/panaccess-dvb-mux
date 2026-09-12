package mpegts

import "fmt"

const (
	PacketSize = 188
	SyncByte   = 0x47
	NullPID    = 0x1FFF
	PATPID     = 0x0000
	CATPID     = 0x0001
	TSDTPID    = 0x0002
	NITPID     = 0x0010
	SDTPID     = 0x0011
	TDTTPID    = 0x0014
)

type Packet struct {
	Data [PacketSize]byte
}

func ParsePacket(b []byte) (Packet, error) {
	var p Packet
	if len(b) != PacketSize {
		return p, fmt.Errorf("invalid TS packet length: %d", len(b))
	}
	if b[0] != SyncByte {
		return p, fmt.Errorf("invalid sync byte: 0x%02x", b[0])
	}
	copy(p.Data[:], b)
	return p, nil
}

func (p Packet) PID() uint16 { return uint16(p.Data[1]&0x1f)<<8 | uint16(p.Data[2]) }
func (p Packet) PUSI() bool { return p.Data[1]&0x40 != 0 }
func (p Packet) TransportPriority() bool { return p.Data[1]&0x20 != 0 }
func (p Packet) ScramblingControl() uint8 { return (p.Data[3] >> 6) & 0x03 }
func (p Packet) AdaptationFieldControl() uint8 { return (p.Data[3] >> 4) & 0x03 }
func (p Packet) HasAdaptation() bool { a := p.AdaptationFieldControl(); return a == 2 || a == 3 }
func (p Packet) HasPayload() bool { a := p.AdaptationFieldControl(); return a == 1 || a == 3 }
func (p Packet) ContinuityCounter() uint8 { return p.Data[3] & 0x0f }

func (p Packet) PayloadOffset() (int, error) {
	if !p.HasPayload() { return PacketSize, nil }
	off := 4
	if p.HasAdaptation() {
		if off >= PacketSize { return 0, fmt.Errorf("invalid adaptation field") }
		n := int(p.Data[4])
		off += 1 + n
		if off > PacketSize { return 0, fmt.Errorf("adaptation field exceeds packet") }
	}
	return off, nil
}

func (p *Packet) SetScramblingControl(v uint8) {
	p.Data[3] = (p.Data[3] & 0x3f) | ((v & 0x03) << 6)
}

func (p Packet) Payload() ([]byte, error) {
	off, err := p.PayloadOffset()
	if err != nil { return nil, err }
	if off == PacketSize { return nil, nil }
	return p.Data[off:], nil
}
