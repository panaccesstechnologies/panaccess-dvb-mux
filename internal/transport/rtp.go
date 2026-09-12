package transport

import (
	"encoding/binary"
	"fmt"
)

// RTPPacket is the parsed RTP header plus its payload. Only the header fields
// needed by MPEG-TS over RTP are retained.
type RTPPacket struct {
	Version        uint8
	Padding        bool
	Extension      bool
	CSRCCount      uint8
	Marker         bool
	PayloadType    uint8
	SequenceNumber uint16
	Timestamp      uint32
	SSRC           uint32
	Payload        []byte
}

// ParseRTP parses an RFC 3550 RTP packet. The payload is not copied.
func ParseRTP(b []byte) (RTPPacket, error) {
	var p RTPPacket
	if len(b) < 12 {
		return p, fmt.Errorf("RTP packet too short: %d", len(b))
	}
	p.Version = b[0] >> 6
	if p.Version != 2 {
		return p, fmt.Errorf("unsupported RTP version %d", p.Version)
	}
	p.Padding = b[0]&0x20 != 0
	p.Extension = b[0]&0x10 != 0
	p.CSRCCount = b[0] & 0x0f
	p.Marker = b[1]&0x80 != 0
	p.PayloadType = b[1] & 0x7f
	p.SequenceNumber = binary.BigEndian.Uint16(b[2:4])
	p.Timestamp = binary.BigEndian.Uint32(b[4:8])
	p.SSRC = binary.BigEndian.Uint32(b[8:12])

	off := 12 + int(p.CSRCCount)*4
	if off > len(b) {
		return p, fmt.Errorf("RTP CSRC list exceeds packet")
	}
	if p.Extension {
		if off+4 > len(b) {
			return p, fmt.Errorf("RTP extension header truncated")
		}
		extLen := int(binary.BigEndian.Uint16(b[off+2:off+4])) * 4
		off += 4 + extLen
		if off > len(b) {
			return p, fmt.Errorf("RTP extension exceeds packet")
		}
	}
	end := len(b)
	if p.Padding {
		pad := int(b[len(b)-1])
		if pad == 0 || pad > end-off {
			return p, fmt.Errorf("invalid RTP padding length %d", pad)
		}
		end -= pad
	}
	p.Payload = b[off:end]
	return p, nil
}

func ValidateMPEGTSRTP(p RTPPacket) error {
	if len(p.Payload) == 0 || len(p.Payload)%188 != 0 {
		return fmt.Errorf("RTP MPEG-TS payload length %d is not a multiple of 188", len(p.Payload))
	}
	return nil
}

// RTPSequenceTracker reports sequence gaps. A duplicate is not reported as a
// forward gap and the tracker resynchronizes after every observation.
type RTPSequenceTracker struct {
	last  uint16
	valid bool
}

func (t *RTPSequenceTracker) Observe(seq uint16) (gap uint16, duplicate bool) {
	if !t.valid {
		t.last, t.valid = seq, true
		return 0, false
	}
	expected := t.last + 1
	if seq == t.last {
		return 0, true
	}
	if seq != expected {
		gap = seq - expected
	}
	t.last = seq
	return gap, false
}
