package service

import (
	"fmt"

	"github.com/panaccesstechnologies/panaccess-dvb-mux/internal/mpegts"
)

// CAEndpoint describes one CA stream signalled by a CA descriptor.
type CAEndpoint struct {
	CASystemID  uint16
	PID         uint16
	PrivateData []byte
}

// Service is the normalized control-plane model used by routing, SimulCrypt,
// ECM/EMM and scrambling layers.
type Service struct {
	ServiceID uint16
	PMTPID    uint16
	PCRPID    uint16
	Streams   []mpegts.PMTStream
	ECM       []CAEndpoint
	EMM       []CAEndpoint
}

// Discover builds a service model from one PAT program, its PMT and the CAT.
// It does not modify PSI and does not generate ECM/EMM payloads.
func Discover(program mpegts.PATProgram, pmt mpegts.PMT, cat mpegts.CAT) (Service, error) {
	if program.Number == 0 { return Service{}, fmt.Errorf("PAT network entry cannot describe a service") }
	if pmt.ProgramNumber != program.Number { return Service{}, fmt.Errorf("PAT program %d does not match PMT program %d", program.Number, pmt.ProgramNumber) }

	var ecm []CAEndpoint
	add := func(ds []mpegts.CADescriptor) {
		for _, d := range ds {
			ecm = append(ecm, CAEndpoint{CASystemID: d.CASystemID, PID: d.CAPID, PrivateData: append([]byte(nil), d.PrivateData...)})
		}
	}

	ds, err := mpegts.ParseCADescriptors(pmt.ProgramDescriptors)
	if err != nil { return Service{}, fmt.Errorf("PMT program CA descriptors: %w", err) }
	add(ds)
	for _, st := range pmt.Streams {
		ds, err := mpegts.ParseCADescriptors(st.Descriptors)
		if err != nil { return Service{}, fmt.Errorf("PMT stream PID 0x%04x CA descriptors: %w", st.PID, err) }
		add(ds)
	}

	ds, err = mpegts.ParseCADescriptors(cat.Descriptors)
	if err != nil { return Service{}, fmt.Errorf("CAT CA descriptors: %w", err) }
	var emm []CAEndpoint
	for _, d := range ds {
		emm = append(emm, CAEndpoint{CASystemID: d.CASystemID, PID: d.CAPID, PrivateData: append([]byte(nil), d.PrivateData...)})
	}

	return Service{ServiceID: program.Number, PMTPID: program.PMTPID, PCRPID: pmt.PCRPID, Streams: append([]mpegts.PMTStream(nil), pmt.Streams...), ECM: ecm, EMM: emm}, nil
}
