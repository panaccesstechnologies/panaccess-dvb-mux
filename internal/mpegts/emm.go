package mpegts

import "fmt"

// EMMInjector packetizes EMM sections/datagrams onto the configured EMM PID
// while maintaining an independent MPEG-TS continuity counter for that PID.
type EMMInjector struct {
	PID uint16
	CC  uint8
}

func NewEMMInjector(pid uint16, cc uint8) (*EMMInjector, error) {
	if pid > 0x1fff {
		return nil, fmt.Errorf("invalid EMM PID")
	}
	return &EMMInjector{PID: pid, CC: cc & 0x0f}, nil
}

// InjectSection converts one EMM section/datagram into TS packets and advances
// the EMM PID continuity counter. No payload scrambling is performed here.
func (e *EMMInjector) InjectSection(section []byte) ([]Packet, error) {
	if e == nil {
		return nil, fmt.Errorf("nil EMM injector")
	}
	packets, err := PacketizeSection(e.PID, section, e.CC)
	if err != nil {
		return nil, err
	}
	e.CC = (e.CC + uint8(len(packets))) & 0x0f
	return packets, nil
}

// InjectTSPacket accepts an ETSI section_TSpkt_flag=0x01 datagram.
// The EMMG/PDG supplies a complete 188-byte MPEG-2 TS packet and the
// head-end is responsible for filling only the PID field. The remaining
// TS header, adaptation field and payload are preserved verbatim.
func (e *EMMInjector) InjectTSPacket(datagram []byte) ([]Packet, error) {
	if e == nil {
		return nil, fmt.Errorf("nil EMM injector")
	}
	if len(datagram) != PacketSize {
		return nil, fmt.Errorf("invalid EMM TS packet length: %d", len(datagram))
	}
	p, err := ParsePacket(datagram)
	if err != nil {
		return nil, fmt.Errorf("invalid EMM TS packet: %w", err)
	}
	p.Data[1] = (p.Data[1] & 0xe0) | byte(e.PID>>8)&0x1f
	p.Data[2] = byte(e.PID)
	return []Packet{p}, nil
}
