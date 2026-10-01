package service

import (
	"testing"
	"github.com/panaccesstechnologies/panaccess-dvb-mux/internal/mpegts"
)

func TestDiscoverServiceCA(t *testing.T) {
	patProgram := mpegts.PATProgram{Number: 101, PMTPID: 0x0100}
	pmt := mpegts.PMT{ProgramNumber: 101, PCRPID: 0x0200, ProgramDescriptors: []byte{0x09, 0x04, 0x4a, 0xfc, 0x1f, 0xfe}, Streams: []mpegts.PMTStream{{StreamType: 0x0f, PID: 0x0200}, {StreamType: 0x1b, PID: 0x0201}}}
	cat := mpegts.CAT{Version: 0, Current: true, Descriptors: []byte{0x09, 0x04, 0x4a, 0xfc, 0x1f, 0xfe}}

	svc, err := Discover(patProgram, pmt, cat)
	if err != nil { t.Fatal(err) }
	if svc.ServiceID != 101 || svc.PMTPID != 0x0100 || svc.PCRPID != 0x0200 { t.Fatalf("unexpected service: %+v", svc) }
	if len(svc.ECM) != 1 || svc.ECM[0].CASystemID != 0x4afc || svc.ECM[0].PID != 0x1ffe { t.Fatalf("unexpected ECM model: %+v", svc.ECM) }
	if len(svc.EMM) != 1 || svc.EMM[0].CASystemID != 0x4afc || svc.EMM[0].PID != 0x1ffe { t.Fatalf("unexpected EMM model: %+v", svc.EMM) }
}
