package simulcrypt

import (
	"bytes"
	"testing"
)

func TestECMGMessageRoundTrip(t *testing.T) {
	in := NewMessage(
		MsgChannelSetup,
		Uint16Parameter(ParamECMChannelID, 7),
		Uint32Parameter(ParamSuperCASID, 0x4afc0001),
	)
	wire, err := Encode(in)
	if err != nil { t.Fatal(err) }

	want := []byte{
		0x03, 0x00, 0x01, 0x00, 0x0e,
		0x00, 0x0e, 0x00, 0x02, 0x00, 0x07,
		0x00, 0x01, 0x00, 0x04, 0x4a, 0xfc, 0x00, 0x01,
	}
	if !bytes.Equal(wire, want) { t.Fatalf("wire=%x want=%x", wire, want) }

	out, err := Decode(wire)
	if err != nil { t.Fatal(err) }
	if out.ProtocolVersion != ECMGProtocolVersion || out.Type != MsgChannelSetup || len(out.Parameters) != 2 {
		t.Fatalf("decoded=%+v", out)
	}
	v, err := Uint16Value(out.Parameters[0])
	if err != nil || v != 7 { t.Fatalf("channel id=%d err=%v", v, err) }
}

func TestECMGDecodeRejectsLengthMismatch(t *testing.T) {
	_, err := Decode([]byte{0x03, 0x00, 0x01, 0x00, 0x04, 0x00, 0x0e, 0x00, 0x02, 0x00})
	if err == nil { t.Fatal("expected length error") }
}

func TestECMGUnknownParameterPreserved(t *testing.T) {
	in := NewMessage(MsgStreamTest, BytesParameter(0x8fff, []byte{1, 2, 3}))
	wire, err := Encode(in)
	if err != nil { t.Fatal(err) }
	out, err := Decode(wire)
	if err != nil { t.Fatal(err) }
	v, ok := out.First(0x8fff)
	if !ok || !bytes.Equal(v, []byte{1, 2, 3}) { t.Fatalf("unknown parameter lost: %x", v) }
}
