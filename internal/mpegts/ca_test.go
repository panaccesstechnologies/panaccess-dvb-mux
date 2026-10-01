package mpegts

import (
	"bytes"
	"testing"
)

func TestCADescriptorParsing(t *testing.T) {
	loop := []byte{0x09, 0x06, 0x4a, 0xfc, 0xff, 0xfe, 0x12, 0x34}
	ds, err := ParseCADescriptors(loop)
	if err != nil { t.Fatal(err) }
	if len(ds) != 1 || ds[0].CASystemID != 0x4afc || ds[0].CAPID != 0x1ffe { t.Fatalf("unexpected CA descriptor: %+v", ds) }
	if !bytes.Equal(ds[0].PrivateData, []byte{0x12, 0x34}) { t.Fatalf("private data mismatch: %x", ds[0].PrivateData) }
}

func TestCADescriptorSkipsUnknown(t *testing.T) {
	loop := []byte{0x52, 0x01, 0x99, 0x09, 0x04, 0x4a, 0xfc, 0x1f, 0xfe}
	ds, err := ParseCADescriptors(loop)
	if err != nil { t.Fatal(err) }
	if len(ds) != 1 || ds[0].CASystemID != 0x4afc || ds[0].CAPID != 0x1ffe { t.Fatalf("unexpected result: %+v", ds) }
}

func TestCATRoundTrip(t *testing.T) {
	in := CAT{Version: 7, Current: true, Descriptors: []byte{0x09, 0x04, 0x4a, 0xfc, 0x1f, 0xfe}}
	s, err := in.MarshalSection()
	if err != nil { t.Fatal(err) }
	out, err := ParseCAT(s)
	if err != nil { t.Fatal(err) }
	if out.Version != in.Version || !out.Current || !bytes.Equal(out.Descriptors, in.Descriptors) { t.Fatalf("CAT mismatch: %+v", out) }
}
