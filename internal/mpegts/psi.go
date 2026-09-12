package mpegts

import (
	"encoding/binary"
	"fmt"
)

type PATProgram struct {
	Number     uint16
	PMTPID     uint16
	NetworkPID uint16 // Used only when Number == 0.
}

type PAT struct {
	TSID     uint16
	Version  uint8
	Current  bool
	Programs []PATProgram
}

type PMTStream struct {
	StreamType  uint8
	PID         uint16
	Descriptors []byte
}

type PMT struct {
	ProgramNumber     uint16
	Version           uint8
	Current           bool
	PCRPID            uint16
	ProgramDescriptors []byte
	Streams            []PMTStream
}

func sectionHeader(tableID byte, ext uint16, version uint8, current bool, payloadLen int) []byte {
	secLen := 5 + payloadLen + 4
	b := make([]byte, 8)
	b[0] = tableID
	b[1] = 0xB0 | byte(secLen>>8)
	b[2] = byte(secLen)
	binary.BigEndian.PutUint16(b[3:5], ext)
	b[5] = 0xC1 | ((version & 0x1f) << 1)
	if !current {
		b[5] &^= 0x01
	}
	b[6], b[7] = 0, 0
	return b
}

func (p PAT) MarshalSection() ([]byte, error) {
	if len(p.Programs) == 0 {
		return nil, fmt.Errorf("PAT must contain at least one program")
	}
	if len(p.Programs)*4 > 1012 {
		return nil, fmt.Errorf("PAT too large")
	}

	payload := make([]byte, 0, 4*len(p.Programs))
	for _, x := range p.Programs {
		if x.Number == 0 {
			if x.NetworkPID > 0x1fff {
				return nil, fmt.Errorf("invalid NIT network PID")
			}
			payload = append(payload, 0, 0, 0xe0|byte(x.NetworkPID>>8), byte(x.NetworkPID))
			continue
		}
		if x.PMTPID > 0x1fff {
			return nil, fmt.Errorf("invalid PMT PID")
		}
		payload = append(payload, byte(x.Number>>8), byte(x.Number), 0xe0|byte(x.PMTPID>>8), byte(x.PMTPID))
	}
	b := sectionHeader(0x00, p.TSID, p.Version, p.Current, len(payload))
	b = append(b, payload...)
	crc := CRC32MPEG(b)
	var c [4]byte
	binary.BigEndian.PutUint32(c[:], crc)
	b = append(b, c[:]...)
	return b, nil
}

func ParsePAT(s []byte) (PAT, error) {
	var p PAT
	if len(s) < 12 || s[0] != 0x00 {
		return p, fmt.Errorf("invalid PAT section")
	}
	if int(s[1]&0x0f)<<8|int(s[2]) != len(s)-3 {
		return p, fmt.Errorf("PAT section length mismatch")
	}
	if !validCRC(s) {
		return p, fmt.Errorf("PAT CRC mismatch")
	}
	p.TSID = binary.BigEndian.Uint16(s[3:5])
	p.Version = (s[5] >> 1) & 0x1f
	p.Current = s[5]&1 != 0
	for i := 8; i+4 <= len(s)-4; i += 4 {
		number := binary.BigEndian.Uint16(s[i : i+2])
		pid := uint16(s[i+2]&0x1f)<<8 | uint16(s[i+3])
		entry := PATProgram{Number: number}
		if number == 0 {
			entry.NetworkPID = pid
		} else {
			entry.PMTPID = pid
		}
		p.Programs = append(p.Programs, entry)
	}
	return p, nil
}

func (p PMT) MarshalSection() ([]byte, error) {
	if p.PCRPID > 0x1fff {
		return nil, fmt.Errorf("invalid PCR PID")
	}
	if len(p.ProgramDescriptors) > 0x0fff || len(p.Streams) > 0xff {
		return nil, fmt.Errorf("PMT too large")
	}
	payload := make([]byte, 0, 4+len(p.ProgramDescriptors)+5*len(p.Streams))
	payload = append(payload, 0xe0|byte(p.PCRPID>>8), byte(p.PCRPID))
	payload = append(payload, 0xf0|byte(len(p.ProgramDescriptors)>>8), byte(len(p.ProgramDescriptors)))
	payload = append(payload, p.ProgramDescriptors...)
	for _, st := range p.Streams {
		if st.PID > 0x1fff || len(st.Descriptors) > 0x0fff {
			return nil, fmt.Errorf("invalid PMT stream")
		}
		payload = append(payload, st.StreamType, 0xe0|byte(st.PID>>8), byte(st.PID), 0xf0|byte(len(st.Descriptors)>>8), byte(len(st.Descriptors)))
		payload = append(payload, st.Descriptors...)
	}
	if 9+len(payload) > 1021 {
		return nil, fmt.Errorf("PMT section exceeds MPEG-2 PSI maximum")
	}
	b := sectionHeader(0x02, p.ProgramNumber, p.Version, p.Current, len(payload))
	b = append(b, payload...)
	crc := CRC32MPEG(b)
	var c [4]byte
	binary.BigEndian.PutUint32(c[:], crc)
	b = append(b, c[:]...)
	return b, nil
}

func ParsePMT(s []byte) (PMT, error) {
	var p PMT
	if len(s) < 16 || s[0] != 0x02 {
		return p, fmt.Errorf("invalid PMT section")
	}
	if int(s[1]&0x0f)<<8|int(s[2]) != len(s)-3 {
		return p, fmt.Errorf("PMT section length mismatch")
	}
	if !validCRC(s) {
		return p, fmt.Errorf("PMT CRC mismatch")
	}
	p.ProgramNumber = binary.BigEndian.Uint16(s[3:5])
	p.Version = (s[5] >> 1) & 0x1f
	p.Current = s[5]&1 != 0
	p.PCRPID = uint16(s[8]&0x1f)<<8 | uint16(s[9])
	pil := int(s[10]&0x0f)<<8 | int(s[11])
	pos := 12
	if pos+pil > len(s)-4 {
		return p, fmt.Errorf("PMT program info overflow")
	}
	p.ProgramDescriptors = append([]byte(nil), s[pos:pos+pil]...)
	pos += pil
	for pos < len(s)-4 {
		if pos+5 > len(s)-4 {
			return p, fmt.Errorf("PMT stream truncated")
		}
		st := PMTStream{StreamType: s[pos], PID: uint16(s[pos+1]&0x1f)<<8 | uint16(s[pos+2])}
		dl := int(s[pos+3]&0x0f)<<8 | int(s[pos+4])
		pos += 5
		if pos+dl > len(s)-4 {
			return p, fmt.Errorf("PMT descriptor overflow")
		}
		st.Descriptors = append([]byte(nil), s[pos:pos+dl]...)
		pos += dl
		p.Streams = append(p.Streams, st)
	}
	return p, nil
}

// PacketizeSection creates 188-byte PSI packets for one section. It is intended
// for PAT/PMT/control-plane injection; a later scheduler will handle repetition.
func PacketizeSection(pid uint16, section []byte, cc uint8) ([]Packet, error) {
	if pid > 0x1fff || len(section) == 0 || len(section) > 1021 {
		return nil, fmt.Errorf("invalid PSI section")
	}
	var out []Packet
	first := true
	pos := 0
	for pos < len(section) {
		var p Packet
		p.Data[0] = 0x47
		p.Data[1] = byte(pid>>8) & 0x1f
		if first {
			p.Data[1] |= 0x40
		}
		p.Data[2] = byte(pid)
		p.Data[3] = 0x10 | (cc & 0x0f)
		cc = (cc + 1) & 0x0f

		off := 4
		if first {
			p.Data[4] = 0
			off = 5
		}
		n := PacketSize - off
		if n > len(section)-pos {
			n = len(section) - pos
		}
		copy(p.Data[off:off+n], section[pos:pos+n])
		for i := off + n; i < PacketSize; i++ {
			p.Data[i] = 0xff
		}
		pos += n
		out = append(out, p)
		first = false
	}
	return out, nil
}
