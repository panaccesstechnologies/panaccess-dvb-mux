package simulcrypt

import "testing"

func TestEMMGDataProvisionUDPMessage(t *testing.T) {
	d := EMMGDataProvision{
		ClientID: 0x11223344,
		DataChannelID: 1,
		DataStreamID: 2,
		DataID: 0x3344,
		Datagrams: [][]byte{{0x01, 0x02}, {0x03}},
	}
	m := d.UDPMessage()
	if m.Type != EMMGMsgDataProvision {
		t.Fatalf("message type = 0x%04x, want 0x0211", m.Type)
	}
	if _, ok := m.First(EMMGParamDataChannelID); ok {
		t.Fatal("UDP data_provision must not contain data_channel_id")
	}
	if _, ok := m.First(EMMGParamDataStreamID); ok {
		t.Fatal("UDP data_provision must not contain data_stream_id")
	}
	if p, ok := m.First(EMMGParamClientID); !ok || len(p) != 4 {
		t.Fatal("UDP data_provision missing client_id")
	}
	if p, ok := m.First(EMMGParamDataID); !ok || len(p) != 2 {
		t.Fatal("UDP data_provision missing data_id")
	}
	if got := len(parameterValues(m, EMMGParamDatagram)); got != 2 {
		t.Fatalf("datagram count = %d, want 2", got)
	}
}

func TestValidateEMMGDatagramsTS(t *testing.T) {
	ts := make([]byte, 188)
	ts[0] = 0x47
	if err := validateDatagrams(0x01, [][]byte{ts}); err != nil {
		t.Fatalf("valid TS packet rejected: %v", err)
	}
	if err := validateDatagrams(0x01, [][]byte{make([]byte, 187)}); err == nil {
		t.Fatal("187-byte TS packet accepted")
	}
	ts[0] = 0x46
	if err := validateDatagrams(0x01, [][]byte{ts}); err == nil {
		t.Fatal("TS packet with invalid sync byte accepted")
	}
}

func TestEMMGErrorParameterEncoding(t *testing.T) {
	m := NewMessage(
		EMMGMsgChannelError,
		Uint16Parameter(ParamErrorStatus, EMMGErrMissingParameter),
		BytesParameter(ParamErrorInformation, []byte("missing parameter")),
	)
	encoded, err := Encode(m)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := Decode(encoded)
	if err != nil {
		t.Fatal(err)
	}
	raw, ok := decoded.First(ParamErrorStatus)
	if !ok {
		t.Fatal("missing error status")
	}
	status, err := Uint16Value(Parameter{Type: ParamErrorStatus, Value: raw})
	if err != nil {
		t.Fatal(err)
	}
	if status != EMMGErrMissingParameter {
		t.Fatalf("status = 0x%04x, want 0x%04x", status, EMMGErrMissingParameter)
	}
}
