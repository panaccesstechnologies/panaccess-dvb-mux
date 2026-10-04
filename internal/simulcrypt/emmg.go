package simulcrypt

const (
	// EMMG message types (ETSI TS 103 197).
	EMMGMsgChannelSetup    uint16 = 0x0011
	EMMGMsgChannelTest     uint16 = 0x0012
	EMMGMsgChannelStatus   uint16 = 0x0013
	EMMGMsgChannelClose    uint16 = 0x0014
	EMMGMsgChannelError    uint16 = 0x0015
	EMMGMsgStreamSetup     uint16 = 0x0111
	EMMGMsgStreamTest      uint16 = 0x0112
	EMMGMsgStreamStatus    uint16 = 0x0113
	EMMGMsgStreamCloseReq  uint16 = 0x0114
	EMMGMsgStreamCloseResp uint16 = 0x0115
	EMMGMsgStreamError     uint16 = 0x0116
	EMMGMsgStreamBWRequest uint16 = 0x0117
	EMMGMsgStreamBWAlloc   uint16 = 0x0118
	EMMGMsgDataProvision   uint16 = 0x0211
)

const (
	EMMGParamClientID        uint16 = 0x0001
	EMMGParamSectionTSPktFlag uint16 = 0x0002
	EMMGParamDataChannelID   uint16 = 0x0003
	EMMGParamDataStreamID    uint16 = 0x0004
	EMMGParamDatagram        uint16 = 0x0005
	EMMGParamBandwidth       uint16 = 0x0006
	EMMGParamDataType        uint16 = 0x0007
	EMMGParamDataID          uint16 = 0x0008
)

const (
	EMMGDataTypeEMM       byte = 0x00
	EMMGDataTypePrivate   byte = 0x01
)

type EMMGDataProvision struct {
	ClientID      uint32
	DataChannelID uint16
	DataStreamID  uint16
	DataID        uint16
	Datagrams     [][]byte
}

func (d EMMGDataProvision) Message() Message {
	params := []Parameter{
		Uint32Parameter(EMMGParamClientID, d.ClientID),
		Uint16Parameter(EMMGParamDataChannelID, d.DataChannelID),
		Uint16Parameter(EMMGParamDataStreamID, d.DataStreamID),
		Uint16Parameter(EMMGParamDataID, d.DataID),
	}
	for _, datagram := range d.Datagrams {
		params = append(params, BytesParameter(EMMGParamDatagram, datagram))
	}
	return NewMessage(EMMGMsgDataProvision, params...)
}

    
// UDPMessage returns an ETSI EMMG data_provision message for UDP transport.
// UDP data_provision contains client_id, data_id and one or more datagrams.
// data_channel_id and data_stream_id are intentionally omitted.
func (d EMMGDataProvision) UDPMessage() Message {
	params := []Parameter{
		Uint32Parameter(EMMGParamClientID, d.ClientID),
		Uint16Parameter(EMMGParamDataID, d.DataID),
	}
	for _, datagram := range d.Datagrams {
		params = append(params, BytesParameter(EMMGParamDatagram, datagram))
	}
	return NewMessage(EMMGMsgDataProvision, params...)
}
