package simulcrypt

import (
	"encoding/binary"
	"fmt"
)

const (
	ECMGProtocolVersion byte = 0x03

	MsgChannelSetup    uint16 = 0x0001
	MsgChannelTest     uint16 = 0x0002
	MsgChannelStatus   uint16 = 0x0003
	MsgChannelClose    uint16 = 0x0004
	MsgChannelError    uint16 = 0x0005
	MsgStreamSetup     uint16 = 0x0101
	MsgStreamTest      uint16 = 0x0102
	MsgStreamStatus    uint16 = 0x0103
	MsgStreamCloseReq  uint16 = 0x0104
	MsgStreamCloseResp uint16 = 0x0105
	MsgStreamError     uint16 = 0x0106
	MsgCWProvision     uint16 = 0x0201
	MsgECMResponse     uint16 = 0x0202
)

const (
	ParamSuperCASID                 uint16 = 0x0001
	ParamSectionTSPktFlag           uint16 = 0x0002
	ParamDelayStart                 uint16 = 0x0003
	ParamDelayStop                  uint16 = 0x0004
	ParamTransitionDelayStart       uint16 = 0x0005
	ParamTransitionDelayStop        uint16 = 0x0006
	ParamECMRepPeriod               uint16 = 0x0007
	ParamMaxStreams                 uint16 = 0x0008
	ParamMinCPDuration              uint16 = 0x0009
	ParamLeadCW                     uint16 = 0x000a
	ParamCWPerMsg                   uint16 = 0x000b
	ParamMaxCompTime                uint16 = 0x000c
	ParamAccessCriteria             uint16 = 0x000d
	ParamECMChannelID               uint16 = 0x000e
	ParamECMStreamID                uint16 = 0x000f
	ParamNominalCPDuration          uint16 = 0x0010
	ParamAccessCriteriaTransferMode uint16 = 0x0011
	ParamCPNumber                   uint16 = 0x0012
	ParamCPDuration                 uint16 = 0x0013
	ParamCPCWCombination            uint16 = 0x0014
	ParamECMDatagram                uint16 = 0x0015
	ParamACDelayStart               uint16 = 0x0016
	ParamACDelayStop                uint16 = 0x0017
	ParamCWEncryption               uint16 = 0x0018
	ParamECMID                      uint16 = 0x0019
	ParamErrorStatus                uint16 = 0x7000
	ParamErrorInformation           uint16 = 0x7001
)

type Parameter struct {
	Type  uint16
	Value []byte
}

type Message struct {
	ProtocolVersion byte
	Type            uint16
	Parameters      []Parameter
}

func NewMessage(messageType uint16, parameters ...Parameter) Message {
	return Message{ProtocolVersion: ECMGProtocolVersion, Type: messageType, Parameters: parameters}
}

func Uint8Parameter(typ uint16, value byte) Parameter {
	return Parameter{Type: typ, Value: []byte{value}}
}

func Uint16Parameter(typ uint16, value uint16) Parameter {
	var b [2]byte
	binary.BigEndian.PutUint16(b[:], value)
	return Parameter{Type: typ, Value: b[:]}
}

func Uint32Parameter(typ uint16, value uint32) Parameter {
	var b [4]byte
	binary.BigEndian.PutUint32(b[:], value)
	return Parameter{Type: typ, Value: b[:]}
}

func BytesParameter(typ uint16, value []byte) Parameter {
	return Parameter{Type: typ, Value: append([]byte(nil), value...)}
}

func Encode(m Message) ([]byte, error) {
	if m.ProtocolVersion == 0 {
		m.ProtocolVersion = ECMGProtocolVersion
	}
	if len(m.Parameters) > 0xffff/4 {
		return nil, fmt.Errorf("too many parameters")
	}
	bodyLen := 0
	for _, p := range m.Parameters {
		if len(p.Value) > 0xffff {
			return nil, fmt.Errorf("parameter 0x%04x too large", p.Type)
		}
		bodyLen += 4 + len(p.Value)
		if bodyLen > 0xffff {
			return nil, fmt.Errorf("message body too large")
		}
	}
	out := make([]byte, 5+bodyLen)
	out[0] = m.ProtocolVersion
	binary.BigEndian.PutUint16(out[1:3], m.Type)
	binary.BigEndian.PutUint16(out[3:5], uint16(bodyLen))
	pos := 5
	for _, p := range m.Parameters {
		binary.BigEndian.PutUint16(out[pos:pos+2], p.Type)
		binary.BigEndian.PutUint16(out[pos+2:pos+4], uint16(len(p.Value)))
		pos += 4
		copy(out[pos:], p.Value)
		pos += len(p.Value)
	}
	return out, nil
}

func Decode(data []byte) (Message, error) {
	var m Message
	if len(data) < 5 {
		return m, fmt.Errorf("ECMG message too short: %d", len(data))
	}
	m.ProtocolVersion = data[0]
	m.Type = binary.BigEndian.Uint16(data[1:3])
	bodyLen := int(binary.BigEndian.Uint16(data[3:5]))
	if bodyLen != len(data)-5 {
		return Message{}, fmt.Errorf("ECMG message length mismatch: header=%d actual=%d", bodyLen, len(data)-5)
	}
	for pos := 5; pos < len(data); {
		if len(data)-pos < 4 {
			return Message{}, fmt.Errorf("parameter header truncated at offset %d", pos)
		}
		typ := binary.BigEndian.Uint16(data[pos : pos+2])
		n := int(binary.BigEndian.Uint16(data[pos+2 : pos+4]))
		pos += 4
		if n > len(data)-pos {
			return Message{}, fmt.Errorf("parameter 0x%04x exceeds message", typ)
		}
		m.Parameters = append(m.Parameters, Parameter{Type: typ, Value: append([]byte(nil), data[pos:pos+n]...)})
		pos += n
	}
	return m, nil
}

func (m Message) First(typ uint16) ([]byte, bool) {
	for _, p := range m.Parameters {
		if p.Type == typ {
			return append([]byte(nil), p.Value...), true
		}
	}
	return nil, false
}

func Uint16Value(p Parameter) (uint16, error) {
	if len(p.Value) != 2 {
		return 0, fmt.Errorf("parameter 0x%04x requires 2 bytes, got %d", p.Type, len(p.Value))
	}
	return binary.BigEndian.Uint16(p.Value), nil
}

func Uint32Value(p Parameter) (uint32, error) {
	if len(p.Value) != 4 {
		return 0, fmt.Errorf("parameter 0x%04x requires 4 bytes, got %d", p.Type, len(p.Value))
	}
	return binary.BigEndian.Uint32(p.Value), nil
}
