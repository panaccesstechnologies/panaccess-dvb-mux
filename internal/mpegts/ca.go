package mpegts

import (
	"encoding/binary"
	"fmt"
)

// CADescriptor is the MPEG-2/DVB conditional-access descriptor (tag 0x09).
type CADescriptor struct {
	CASystemID  uint16
	CAPID       uint16
	PrivateData []byte
}

// ParseCADescriptors parses a descriptor loop and returns all CA descriptors.
func ParseCADescriptors(loop []byte) ([]CADescriptor, error) {
	var out []CADescriptor
	for pos := 0; pos < len(loop); {
		if len(loop)-pos < 2 {
			return nil, fmt.Errorf("descriptor header truncated at offset %d", pos)
		}
		tag := loop[pos]
		length := int(loop[pos+1])
		pos += 2
		if length > len(loop)-pos {
			return nil, fmt.Errorf("descriptor tag 0x%02x length %d exceeds loop", tag, length)
		}
		data := loop[pos : pos+length]
		if tag == 0x09 {
			if length < 4 {
				return nil, fmt.Errorf("CA descriptor too short: %d", length)
			}
			out = append(out, CADescriptor{
				CASystemID:  binary.BigEndian.Uint16(data[:2]),
				CAPID:       uint16(data[2]&0x1f)<<8 | uint16(data[3]),
				PrivateData: append([]byte(nil), data[4:]...),
			})
		}
		pos += length
	}
	return out, nil
}

// FirstCADescriptor returns the first CA descriptor in a descriptor loop.
func FirstCADescriptor(loop []byte) (CADescriptor, bool, error) {
	ds, err := ParseCADescriptors(loop)
	if err != nil || len(ds) == 0 {
		return CADescriptor{}, false, err
	}
	return ds[0], true, nil
}

// CAT is the MPEG-2 Conditional Access Table (table_id 0x01).
type CAT struct {
	Version     uint8
	Current     bool
	Descriptors []byte
}

// MarshalSection creates a complete CAT section including CRC32/MPEG.
func (c CAT) MarshalSection() ([]byte, error) {
	if len(c.Descriptors) > 0x0fff {
		return nil, fmt.Errorf("CAT descriptors too large")
	}
	b := sectionHeader(0x01, 0, c.Version, c.Current, len(c.Descriptors))
	b = append(b, c.Descriptors...)
	crc := CRC32MPEG(b)
	var sum [4]byte
	binary.BigEndian.PutUint32(sum[:], crc)
	b = append(b, sum[:]...)
	return b, nil
}

// ParseCAT parses a complete CAT section and validates its CRC.
func ParseCAT(s []byte) (CAT, error) {
	var c CAT
	if len(s) < 12 || s[0] != 0x01 {
		return c, fmt.Errorf("invalid CAT section")
	}
	if int(s[1]&0x0f)<<8|int(s[2]) != len(s)-3 {
		return c, fmt.Errorf("CAT section length mismatch")
	}
	if !validCRC(s) {
		return c, fmt.Errorf("CAT CRC mismatch")
	}
	c.Version = (s[5] >> 1) & 0x1f
	c.Current = s[5]&1 != 0
	c.Descriptors = append([]byte(nil), s[8:len(s)-4]...)
	if _, err := ParseCADescriptors(c.Descriptors); err != nil {
		return CAT{}, err
	}
	return c, nil
}
