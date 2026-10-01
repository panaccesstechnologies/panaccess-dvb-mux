package service

import (
	"fmt"

	"github.com/panaccesstechnologies/panaccess-dvb-mux/internal/mpegts"
)

// EMMInserter packetizes EMM datagrams onto the EMM PID discovered from CAT.
type EMMInserter struct {
	Endpoint CAEndpoint
	Injector *mpegts.EMMInjector
}

func NewEMMInserter(s Service, caSystemID uint16, initialCC uint8) (*EMMInserter, error) {
	for _, ep := range s.EMM {
		if ep.CASystemID == caSystemID {
			inj, err := mpegts.NewEMMInjector(ep.PID, initialCC)
			if err != nil { return nil, err }
			return &EMMInserter{Endpoint: ep, Injector: inj}, nil
		}
	}
	return nil, fmt.Errorf("EMM CA system 0x%04x not found", caSystemID)
}

// InjectDatagram inserts one EMM section/datagram returned by an EMMG data
// provision path. EMMG datagrams are expected to be MPEG-2 sections when
// section_TSpkt_flag is configured for sections.
func (e *EMMInserter) InjectDatagram(datagram []byte) ([]mpegts.Packet, error) {
	if e == nil || e.Injector == nil { return nil, fmt.Errorf("nil EMM inserter") }
	if len(datagram) == 0 { return nil, fmt.Errorf("empty EMM datagram") }
	return e.Injector.InjectSection(datagram)
}
